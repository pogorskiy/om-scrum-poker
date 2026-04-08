# Архитектурный обзор: om-scrum-poker

**Дата:** 2026-04-08
**Архитектор:** Claude Opus 4.6
**Версия кодовой базы:** commit f95ea54 (main)
**Область:** полный стек (Go backend + Preact frontend + инфраструктура)

---

## Находки

### 1. [75] Двухуровневая блокировка с риском deadlock между RoomManager.mu и Room.mu

**Категория:** Concurrency / Mutex patterns
**Серьёзность:** HIGH

**Описание:**
В системе существуют два уровня мьютексов: `RoomManager.mu` (RWMutex) и `Room.mu` (Mutex). Во всех обработчиках WebSocket-событий паттерн следующий:

1. `manager.GetRoom(roomID)` -- захватывает `rm.mu.RLock()`/`RUnlock()`
2. `room.Lock()` -- захватывает `room.mu`
3. Мутация состояния
4. `room.Unlock()`
5. `manager.Broadcast()` -- снова захватывает `rm.mu.RLock()`

Сам по себе этот паттерн безопасен (lock ordering соблюдается). Однако проблема в `collectGarbage()`:

```go
func (rm *RoomManager) collectGarbage() {
    rm.mu.Lock()  // exclusive lock на весь менеджер
    defer rm.mu.Unlock()
    // ... итерация по всем комнатам
}
```

GC берёт эксклюзивный `rm.mu.Lock()`, что блокирует ВСЕ операции с комнатами (включая `GetRoom`, `Broadcast`, `RegisterClient`) на время итерации. При большом числе комнат (даже сотни) это создаёт ощутимую задержку для всех пользователей. Более того, `collectGarbage` обращается к `room.GetLastActivity()` (atomic, без room.Lock), но сверяет это с `len(clients)` под `rm.mu.Lock` -- потенциально удаляя комнату, в которой в данный момент обрабатывается join.

**Рекомендация:**
1. Разделить GC на две фазы: сначала собрать кандидатов под `RLock`, затем удалять по одному под `Lock` с повторной проверкой.
2. Рассмотреть `sync.Map` для `rooms` или шардирование по roomID.

---

### 2. [72] Отсутствие backpressure на уровне Broadcast -- медленный клиент блокирует отправку остальным

**Категория:** WebSocket / Fan-out
**Серьёзность:** HIGH

**Описание:**
Метод `Broadcast` в `room_manager.go` отправляет сообщения клиентам последовательно:

```go
for _, c := range list {
    c.Send(msg)
}
```

`Client.Send()` -- неблокирующий (select с default), при переполнении буфера закрывает соединение. Это хорошо. Однако если у нескольких клиентов буфер почти полон, Broadcast будет вызывать `Close()` для каждого из них последовательно. При этом `Close()` закрывает канал `done`, что вызывает завершение `WritePump`, который в это время может писать в сокет с таймаутом 5 секунд.

При комнате с 50 участниками (максимум) один медленный клиент не задержит остальных благодаря буферу в 32 сообщения. Но каскадный сценарий (сервер под нагрузкой, много клиентов отстают одновременно) может создать пик вызовов `Close()`.

**Рекомендация:**
Отправлять broadcast параллельно через горутины или worker pool. Текущая реализация приемлема для масштаба проекта (50 участников), но не масштабируется.

---

### 3. [68] Race condition при disconnect: broadcast "disconnected" после Unregister

**Категория:** Concurrency / WebSocket lifecycle
**Серьёзность:** MEDIUM

**Описание:**
В `HandleWebSocket` (ws.go, строки 99-123) после выхода из `readPump`:

```go
client.Close()
manager.UnregisterClient(roomID, client)  // клиент удалён из списка

sid := client.SessionID()
if sid != "" {
    room := manager.GetRoom(roomID)
    if room != nil {
        room.Lock()
        if p, ok := room.Participants[sid]; ok {
            p.Status = "disconnected"
        }
        room.Unlock()
        // Broadcast -- но client уже unregistered!
        manager.Broadcast(roomID, msg)
    }
}
```

Порядок операций корректен: клиент отписан ДО broadcast, поэтому не получит собственное уведомление. Однако существует окно между `UnregisterClient` и установкой `Status = "disconnected"`: если в этот момент другой клиент запросит `room_state` (через rejoin), он увидит участника со статусом "active", хотя тот уже отключён.

**Рекомендация:**
Переместить установку `Status = "disconnected"` и broadcast ДО `UnregisterClient`, или объединить их в одну атомарную операцию.

---

### 4. [65] Room.Participants как публичное поле -- нарушение инкапсуляции домена

**Категория:** Package structure / Encapsulation
**Серьёзность:** MEDIUM

**Описание:**
`Room.Participants` -- это `map[string]*Participant`, экспортируемое публичное поле. Серверный пакет напрямую читает и модифицирует его:

- `ws.go:108`: `p.Status = "disconnected"` -- прямая мутация без метода Room
- `ws.go:411`: `room.Participants[client.SessionID()]` -- прямой доступ к карте
- `room_manager.go:153`: `p.LastPing = time.Now()` -- прямая мутация
- `room_manager.go:221-229`: итерация по Participants для BuildRoomState

Это означает, что серверный пакет знает внутреннюю структуру домена и может обойти валидацию (например, установить невалидный статус). При рефакторинге домена придётся менять и серверный пакет.

**Рекомендация:**
Добавить методы в `Room`: `SetParticipantStatus(sid, status)`, `GetParticipantSnapshot()`, `UpdatePingTime(sid)`. Это позволит домену контролировать инварианты.

---

### 5. [62] Timer: нет серверного уведомления при истечении -- клиенты управляют переходом expired

**Категория:** Client-server contract
**Серьёзность:** MEDIUM

**Описание:**
Таймер работает по принципу "ленивого вычисления": сервер НЕ запускает горутину для обратного отсчёта. Переход running -> expired происходит ТОЛЬКО при вызове `TimerInfo()` (строка 436-444 в room.go), который вызывается при следующем обращении.

На клиенте (Timer.tsx) переход в expired делается локально:
```typescript
if (next <= 0) {
    timerState.value = { ...timerState.value, state: 'expired', remaining: 0 };
    clearInterval(id);
    return 0;
}
```

Это означает:
1. Если никто не обращается к комнате, таймер технически "бесконечно running" на сервере.
2. Разные клиенты могут показать expired в разные моменты (расхождение часов).
3. Нет серверного события `timer_expired`, на которое можно повесить логику (авто-reveal и т.п.).

**Рекомендация:**
Для текущих требований (таймер декоративный) это приемлемо. Но если планируется авто-reveal по таймеру, потребуется серверная горутина с `time.AfterFunc`.

---

### 6. [58] Монолитный пакет `internal/server` -- все обработчики в одном пакете

**Категория:** Package structure / Cohesion
**Серьёзность:** MEDIUM

**Описание:**
Пакет `internal/server` содержит 9 файлов (не считая тестов):
- `handler.go` (209 строк) -- HTTP-сервер + SPA
- `ws.go` (636 строк!) -- WebSocket lifecycle + ВСЕ обработчики событий (join, vote, reveal, timer, leave, etc.)
- `room_manager.go` (252 строки) -- управление комнатами + broadcast
- `client.go` (128 строк) -- WebSocket-клиент
- `events.go` (135 строк) -- типы сообщений
- `conntracker.go`, `ratelimit.go`, `msg_rate_limiter.go` -- rate limiting

Файл `ws.go` -- самый проблемный: 636 строк с 13 handler-функциями. Каждая функция следует одному и тому же паттерну (проверка sessionID -> получение комнаты -> Lock -> мутация -> Unlock -> broadcast), что создаёт много boilerplate.

**Рекомендация:**
1. Вынести обработчики событий в отдельный файл `event_handlers.go`.
2. Рассмотреть middleware-подход для повторяющихся проверок (sessionID, room existence).
3. Для KISS-проекта текущее состояние терпимо, но при добавлении новых событий `ws.go` станет неуправляемым.

---

### 7. [55] Отсутствие CORS-заголовков для /health эндпоинта

**Категория:** HTTP / Security
**Серьёзность:** MEDIUM

**Описание:**
Эндпоинт `/health` не устанавливает CORS-заголовки. Frontend вызывает его через `fetch('/health')` в компоненте Footer (Footer.tsx:26). Это работает в production (same-origin), но при разработке с Vite (порт 5173 -> 8080) запрос `/health` проксируется через Vite dev server.

Однако если кто-то захочет мониторить `/health` с внешнего домена -- это не сработает. Более важно: при текущей архитектуре `ALLOWED_ORIGINS` влияет ТОЛЬКО на WebSocket accept, но не на HTTP-эндпоинты.

**Рекомендация:**
Добавить CORS middleware для HTTP-эндпоинтов, либо документировать, что `/health` доступен только same-origin.

---

### 8. [52] Фронтенд: handleMessage в ws.ts -- гигантский switch без типовой безопасности payload

**Категория:** Frontend architecture / Type safety
**Серьёзность:** MEDIUM

**Описание:**
Функция `handleMessage` в `ws.ts` (строки 78-285) -- это switch на 200+ строк с 13 ветками. Каждая ветка вручную обрабатывает payload и мутирует `roomState.value` через spread-оператор.

Проблемы:
1. Нет валидации payload для каждого типа события (только общая проверка `isValidServerMessage`).
2. Каждая мутация создаёт новый объект через spread -- много аллокаций при частых обновлениях.
3. Паттерн `roomState.value.participants.map(p => p.sessionId === x ? {...p, field: val} : p)` повторяется 6 раз.

**Рекомендация:**
1. Вынести общий хелпер `updateParticipant(sessionId, updater)`.
2. Рассмотреть map из обработчиков вместо switch.
3. Текущий подход работает, но при добавлении новых событий файл станет трудночитаемым.

---

### 9. [50] Docker scratch image без healthcheck и без CA-сертификатов

**Категория:** Build / Deployment
**Серьёзность:** MEDIUM

**Описание:**
Dockerfile использует `FROM scratch` для runtime-образа. Это минимальный размер, но:

1. Нет CA-сертификатов -- если сервер когда-нибудь будет делать исходящие HTTPS-запросы (webhook, OAuth), это сломается.
2. Нет `HEALTHCHECK` директивы -- оркестраторы (Docker Compose, ECS) не смогут проверять здоровье контейнера.
3. Нет shell -- невозможно exec в контейнер для отладки.
4. Нет `USER` directive -- процесс запускается от root (хотя в scratch это менее критично).

**Рекомендация:**
Добавить `HEALTHCHECK --interval=30s CMD ["/om-scrum-poker", "-health"]` (потребует флаг в main.go) или переключиться на `gcr.io/distroless/static` для более безопасного минимального образа.

---

### 10. [48] Клиентский sessionId в localStorage -- нет ротации, нет привязки к устройству

**Категория:** Security / Identity
**Серьёзность:** MEDIUM

**Описание:**
Session ID генерируется один раз (`generateHex(32)`) и хранится в localStorage навсегда. Он используется как единственный идентификатор участника. Проблемы:

1. **Угон сессии:** если кто-то скопирует значение `om-poker-session` из localStorage, он может присоединиться к любой комнате под чужим именем.
2. **Нет экспирации:** sessionId живёт вечно в localStorage.
3. **Нет привязки к вкладке:** две вкладки одного браузера используют один sessionId, что приведёт к конфликту (две WebSocket-соединения с одним sessionId в одной комнате).

Последний пункт -- реальный баг: при открытии комнаты в двух вкладках оба соединения зарегистрируются в `clients` map, но participant будет один. При закрытии одной вкладки сервер пометит участника как "disconnected", хотя вторая вкладка ещё активна.

**Рекомендация:**
1. Использовать per-tab sessionId (sessionStorage вместо localStorage) или добавить tab-specific suffix.
2. Для KISS-проекта без auth это приемлемый компромисс, но стоит документировать ограничение.

---

### 11. [45] BuildRoomState сортирует по sessionId, а не по порядку join

**Категория:** UX Rules / Contract
**Серьёзность:** MEDIUM

**Описание:**
CLAUDE.md явно говорит: "Participant list must NOT be re-sorted by status. Order is by join time (as returned by the server)."

Однако `BuildRoomState` (room_manager.go:231-233) сортирует по sessionId:
```go
sort.Slice(participants, func(i, j int) bool {
    return participants[i].SessionID < participants[j].SessionID
})
```

SessionID -- это случайный hex, поэтому порядок фактически случайный и не соответствует порядку join. `Participants` -- это `map[string]*Participant`, который не сохраняет порядок вставки. Для сохранения порядка join нужен либо slice, либо дополнительное поле `JoinedAt` в Participant.

Комментарий в коде говорит "Sort for deterministic output", что правильно для тестов, но нарушает UX-требование.

**Рекомендация:**
Добавить поле `JoinedAt time.Time` в `Participant` и сортировать по нему. Текущий hex-based порядок стабилен (не прыгает), но не соответствует заявленному в документации.

---

### 12. [42] Нет graceful degradation при потере WebSocket на клиенте -- состояние может устареть

**Категория:** Client-server contract / Resilience
**Серьёзность:** MEDIUM

**Описание:**
При реконнекте клиент отправляет `join` и получает свежий `room_state`. Это хорошо. Однако между disconnect и reconnect могут произойти события (голоса, reveal, new_round), которые клиент пропустит. После reconnect `room_state` всё восстановит.

Но есть тонкость: `messageQueue` в `ws.ts` накапливает сообщения клиента во время disconnect. При reconnect `flushQueue()` отправит их ВСЕ. Если пользователь проголосовал во время disconnect, а за это время начался новый раунд -- его голос применится к новому раунду, что может быть неожиданным.

**Рекомендация:**
Очищать `messageQueue` при получении `room_state` (он уже содержит актуальное состояние), или добавить timestamp/round-number в сообщения для валидации.

---

### 13. [40] Фронтенд: сигнал timerState дублирует данные из roomState

**Категория:** Frontend architecture / State management
**Серьёзность:** MEDIUM

**Описание:**
Есть два источника данных для таймера:
- `roomState.value.timer` (приходит с сервером в `room_state`)
- `timerState` (отдельный сигнал, обновляется как из `room_state`, так и из `timer_updated`)

В `handleMessage` при получении `room_state`:
```typescript
if (msg.payload.timer) {
    timerState.value = msg.payload.timer;
}
```

А Timer.tsx напрямую мутирует `timerState.value` при auto-expire:
```typescript
timerState.value = { ...timerState.value, state: 'expired', remaining: 0 };
```

Два источника истины для одних данных -- классическая проблема десинхронизации.

**Рекомендация:**
Использовать `computed` сигнал для таймера, основанный на `roomState`, или убрать таймер из `roomState` payload и полагаться только на `timer_updated`.

---

### 14. [38] Отсутствие интеграционных тестов WebSocket lifecycle

**Категория:** Testability
**Серьёзность:** LOW

**Описание:**
Unit-тесты в `ws_test.go` используют `fakeClient` (без реального WebSocket), что позволяет тестировать логику обработчиков. Однако нет интеграционных тестов, которые:

1. Устанавливают реальное WebSocket-соединение к тестовому серверу.
2. Проверяют полный lifecycle: connect -> join -> vote -> reveal -> disconnect -> reconnect.
3. Проверяют rate limiting через реальные HTTP-запросы.
4. Проверяют SPA fallback с реальным embed.FS.

Модуль `nhooyr.io/websocket` предоставляет `websockettest` для таких тестов.

**Рекомендация:**
Добавить хотя бы один end-to-end тест с `httptest.Server` + `websocket.Dial`.

---

### 15. [35] Жёсткая привязка card values в фронтенде и бэкенде

**Категория:** Extensibility
**Серьёзность:** LOW

**Описание:**
Набор допустимых голосов захардкожен в двух местах:

- Backend: `domain.ValidVotes` map в `room.go`
- Frontend: `CARD_VALUES` array в `CardDeck.tsx`

Нет протокольного механизма для передачи допустимых значений с сервера. Если кто-то захочет изменить масштаб (T-shirt sizes, Fibonacci без 40 и 100), нужно менять оба файла.

**Рекомендация:**
Для KISS-проекта это нормально. При необходимости -- добавить `validCards` в `room_state` payload.

---

### 16. [32] Makefile не запускает frontend тесты

**Категория:** Build / CI
**Серьёзность:** LOW

**Описание:**
Таргет `test` запускает только Go-тесты:
```makefile
test:
    go test ./...
```

Frontend-тесты (`cd web && npm test`) нужно запускать отдельно. В CI это может привести к тому, что frontend-тесты будут пропущены.

**Рекомендация:**
Добавить `test-all` target:
```makefile
test-all: test
    cd web && npm ci && npm test
```

---

### 17. [30] go.mod указывает Go 1.22, но можно использовать более новые фичи

**Категория:** Dependencies / Maintenance
**Серьёзность:** LOW

**Описание:**
`go.mod` указывает `go 1.22.0`. Единственная внешняя зависимость -- `nhooyr.io/websocket v1.8.17`. Стоит отметить, что v1.8.x -- это legacy-ветка; текущая версия -- v2.x с другим API и лучшей поддержкой контекстов.

**Рекомендация:**
Рассмотреть миграцию на `nhooyr.io/websocket/v2` при следующем крупном рефакторинге. Текущая версия стабильна и работает.

---

### 18. [28] handleClearRoom: broadcast отправляется ДО мутации

**Категория:** Correctness / Event ordering
**Серьёзность:** LOW

**Описание:**
В `handleClearRoom` (ws.go:380-387):
```go
// Broadcast before clearing so all connected clients receive the event.
if msg := makeEnvelopeOrLog("room_cleared", struct{}{}); msg != nil {
    manager.Broadcast(client.roomID, msg)
}
room.Lock()
room.ClearRoom()
room.Unlock()
```

Broadcast отправляется ДО очистки комнаты. Комментарий объясняет intent, но это значит, что если клиент получит `room_cleared` и немедленно отправит `join` (что он делает -- см. ws.ts:229), join может обработаться ДО `ClearRoom()`, и участник будет удалён.

На практике это маловероятно (network latency), но теоретически возможно при loopback-соединении.

**Рекомендация:**
Инвертировать порядок: сначала `ClearRoom()`, потом broadcast. Клиент в любом случае отправит `join` заново.

---

### 19. [25] Логирование через stdlib log без структурирования

**Категория:** Observability
**Серьёзность:** LOW

**Описание:**
Весь проект использует `log.Printf` для логирования. Нет уровней (info/warn/error), нет структурированных полей (roomID, sessionID, IP). При отладке production-инцидента будет трудно фильтровать логи.

**Рекомендация:**
Для KISS-проекта `log.Printf` достаточен. При росте -- рассмотреть `slog` (stdlib с Go 1.21).

---

### 20. [22] Embed стратегия: web/embed.go в отдельном пакете -- хорошее решение

**Категория:** Build / Architecture
**Серьёзность:** NOTE (позитивная находка)

**Описание:**
`web/embed.go` выносит `//go:embed dist` в отдельный пакет `web`, что позволяет:
1. Импортировать `web.DistFS` из main без circular dependency.
2. Держать `.gitkeep` в `dist/` для компиляции без предварительной сборки фронтенда.
3. Чётко разделить frontend и backend в файловой структуре.

Fallback-стратегия (embedded -> disk -> placeholder) в `handleSPA` тоже продумана.

---

### 21. [20] Frontend: отсутствие Error Boundaries

**Категория:** Frontend architecture / Resilience
**Серьёзность:** LOW

**Описание:**
Нет Error Boundary компонента. Если `RoomPage` или `ParticipantList` выбросят исключение при рендеринге (например, из-за некорректного серверного payload), весь UI рухнет без возможности восстановления.

**Рекомендация:**
Добавить ErrorBoundary обёртку вокруг `<RoomPage>`.

---

### 22. [18] Хорошая модульность фронтенд-компонентов

**Категория:** Frontend architecture
**Серьёзность:** NOTE (позитивная находка)

**Описание:**
Компонентная структура чистая:
- Каждый компонент в своей папке (PascalCase) с `.tsx` + `.css` + `.test.tsx`.
- BEM-именование CSS-классов.
- Разделение на presentation (Card, ParticipantCard) и container (RoomPage, HomePage) компоненты.
- Signals для глобального состояния, hooks для побочных эффектов.
- Modal с правильной a11y (dialog element, focus trap, ESC handling).

---

### 23. [15] Хорошо реализованная система rate limiting

**Категория:** Security
**Серьёзность:** NOTE (позитивная находка)

**Описание:**
Трёхуровневая защита:
1. `ConnTracker` -- per-IP и глобальный лимит одновременных соединений.
2. `RateLimiter` -- token bucket для создания комнат и WebSocket-подключений per-IP.
3. `MsgRateLimiter` -- per-connection лимит сообщений (20 msg/sec burst).

Все три уровня имеют cleanup goroutines и тесты. Token bucket реализация корректна.

---

### 24. [12] Reconnect-логика на клиенте -- грамотная реализация

**Категория:** Client-server contract
**Серьёзность:** NOTE (позитивная находка)

**Описание:**
- Exponential backoff с jitter (BASE_DELAY * 2^attempt + 30% random).
- Переход на slow polling (10s) после RECONNECT_TIMEOUT (30s).
- Визуальная индикация через ConnectionBanner (reconnecting / connection lost + retry).
- Автоматический rejoin при reconnect (отправка join в onopen).
- Отправка `leave` перед disconnect для немедленного удаления на сервере.
- Полное покрытие тестами (ws.test.ts).

---

## Сводка

| Уровень | Количество | Ключевые области |
|---------|-----------|-----------------|
| HIGH (70-89) | 2 | Concurrency (GC locking), WebSocket broadcast |
| MEDIUM (40-69) | 11 | Race conditions, encapsulation, timer design, state duplication, UX ordering |
| LOW (20-39) | 6 | Tests, logging, Dockerfile, extensibility, error boundaries |
| NOTE (0-19) | 5 | Позитивные находки: embed, components, rate limiting, reconnect |

**Общая оценка:** Проект хорошо спроектирован для своего масштаба. Принцип KISS соблюдается: минимум зависимостей, простая архитектура, понятная кодовая база. Основные риски лежат в области concurrency (двухуровневая блокировка, race при disconnect) и масштабируемости (последовательный broadcast, монолитный ws.go). Для текущих требований (до 50 участников в комнате, in-memory state) архитектура адекватна.

**Самые приоритетные исправления:**
1. Race condition при disconnect (#3) -- реальный баг.
2. BuildRoomState сортировка (#11) -- нарушение документированного UX-контракта.
3. handleClearRoom event ordering (#18) -- потенциальный race при loopback.
