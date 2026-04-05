# Архитектор — Ревью проекта
- Дата: 2026-04-05
- Статус: done
- Охват: cmd/server/main.go, internal/server/handler.go, internal/server/ws.go, internal/server/room_manager.go, internal/server/client.go, internal/server/events.go, internal/server/ratelimit.go, internal/domain/room.go, internal/domain/stats.go, web/embed.go, web/src/state.ts, web/src/ws.ts, web/src/app.tsx, web/src/hooks/usePresence.ts, web/src/components/RoomPage/RoomPage.tsx, Makefile, Dockerfile, go.mod, docs/planning/04-architecture.md, все *_test.go файлы
- Модель: claude-opus-4-6

---

## [Безопасность]

### [92] Отсутствие авторизации на действия reveal/new_round/clear_room
- **Severity:** CRITICAL
- **Файл:** internal/server/ws.go:249-319
- **Проблема:** Любой участник комнаты может выполнить reveal, new_round или clear_room. Нет понятия владельца комнаты или ролей. Злоумышленник, зная ID комнаты (12 hex = 48 бит), может подключиться и сбросить голосование или очистить комнату целиком.
- **Влияние:** Все пользователи комнаты теряют результаты голосования. При использовании в командах с предсказуемыми roomId (например, "sprint-42-...") — легко эксплуатируемо.
- **Рекомендация:** Ввести минимальную модель ролей: создатель комнаты получает роль "host", только host может делать reveal/clear_room. Или добавить опциональный пароль на комнату. На минимуме — добавить rate limit на эти действия.
- **Effort:** medium

### [90] ~~WebSocket InsecureSkipVerify — отсутствие проверки Origin~~ RESOLVED
- **Severity:** CRITICAL
- **Файл:** internal/server/ws.go:33-35
- **Проблема:** `InsecureSkipVerify: true` в `websocket.AcceptOptions` полностью отключает проверку Origin. Это позволяет CSRF-подобные атаки через WebSocket: вредоносная веб-страница может открыть WebSocket к серверу от имени пользователя.
- **Влияние:** Любой сайт в браузере пользователя может подключиться к его покер-сессии и выполнять действия от его имени.
- **Рекомендация:** Заменить `InsecureSkipVerify` на явный список допустимых Origin. В development-режиме можно оставить `localhost`. Добавить `OriginPatterns` в конфигурацию.
- **Effort:** low
- **Решение:** Реализовано. `InsecureSkipVerify` удалён, добавлена `ALLOWED_ORIGINS` env var.

### [82] SessionID генерируется на клиенте без серверной валидации
- **Severity:** HIGH
- **Файл:** web/src/state.ts:88-94, internal/server/ws.go:131-145
- **Проблема:** SessionID генерируется в браузере через `crypto.getRandomValues` (32 hex символа) и просто передается серверу в `join` payload. Сервер не валидирует формат, длину, не хранит сессии. Любой может передать произвольный sessionID и "захватить" участника с таким ID в комнате.
- **Влияние:** Возможна подмена сессии другого участника (session hijacking). Если злоумышленник узнает sessionID жертвы — он может переподключиться от её имени.
- **Рекомендация:** Генерировать sessionID на сервере при первом подключении (или привязывать к WebSocket connection). Альтернатива — валидировать формат и запретить смену sessionID после первого join в комнате.
- **Effort:** medium

---

## [Архитектура / Масштабируемость]

### [78] Полное отсутствие горизонтального масштабирования
- **Severity:** HIGH
- **Файл:** internal/server/room_manager.go (весь файл)
- **Проблема:** Все состояние хранится в памяти одного процесса. Нет механизма для запуска нескольких инстансов: нет pub/sub для broadcast, нет shared state. При запуске за load balancer — участники одной комнаты могут попасть на разные инстансы и видеть разное состояние.
- **Влияние:** Ограничение на один инстанс. При высокой нагрузке (>1000 одновременных комнат) — единственный вариант масштабирования vertical.
- **Рекомендация:** Для текущего масштаба проекта — это осознанный trade-off (документирован в архитектуре). Если потребуется масштабирование: добавить sticky sessions на уровне load balancer (по roomId), или Redis pub/sub для broadcast между инстансами.
- **Effort:** high

### [72] Потеря состояния при рестарте
- **Severity:** HIGH
- **Файл:** internal/server/room_manager.go:27-31
- **Проблема:** Все комнаты и голосования теряются при рестарте процесса. Нет механизма persistence. При деплое новой версии — все активные сессии обрываются.
- **Влияние:** Пользователи теряют ход голосования при каждом обновлении сервера. Docker restart = потеря данных.
- **Рекомендация:** Для текущего use-case (эфемерные покер-сессии) — приемлемо, задокументировано. При необходимости — добавить optional snapshot в файл при graceful shutdown и восстановление при старте.
- **Effort:** medium

---

## [Расхождения с архитектурной документацией]

### [65] WebSocket URL: архитектура `/ws/room/{roomId}`, реализация `/ws/{roomId}`
- **Severity:** MEDIUM
- **Файл:** internal/server/ws.go:27-28 vs docs/planning/04-architecture.md:372
- **Проблема:** Архитектурная документация (v3.0+) явно указывает URL `/ws/room/{roomId}`, но реализация использует `/ws/{roomId}` (без `/room/` сегмента). Маршрутизатор: `mux.HandleFunc("/ws/", ...)` обрезает `/ws/` prefix. Фронтенд в `ws.ts:27` использует `/ws/${roomId}` (без room).
- **Влияние:** Несоответствие документации и кода. Новый разработчик может быть введен в заблуждение.
- **Рекомендация:** Привести документацию в соответствие с реализацией (убрать `/room/` из URL в docs), либо добавить сегмент `/room/` в реализацию.
- **Effort:** low

### [58] Отсутствие поля `roomName` в `join` payload
- **Severity:** MEDIUM
- **Файл:** internal/server/events.go:18-21 vs docs/planning/04-architecture.md:393
- **Проблема:** Архитектурная документация v3.0 добавила `roomName` в `join` payload (changelog: "Added `roomName` to `join` message payload"). В реализации `JoinPayload` содержит только `sessionId` и `userName`. Имя комнаты генерируется как `userName + "'s Room"` (ws.go:155).
- **Влияние:** Клиент не может задать имя комнаты — оно всегда формируется по шаблону.
- **Рекомендация:** Добавить `RoomName` в `JoinPayload` или обновить документацию, убрав это требование.
- **Effort:** low

### [52] Отсутствие rate limit на WebSocket-сообщения
- **Severity:** MEDIUM
- **Файл:** internal/server/ws.go:83-106, docs/planning/04-architecture.md:348-349
- **Проблема:** Архитектурная документация указывает rate limit "30/second per connection" для inbound WS-сообщений. В реализации `readPump` не содержит никакого rate limiting — только `limiter.AllowWSConnection` при подключении и `limiter.AllowRoomCreation` при join.
- **Влияние:** Злоумышленник может отправлять неограниченное количество сообщений через открытый WebSocket. Потенциальная DoS-атака на server-side dispatch logic.
- **Рекомендация:** Добавить per-connection message rate limiter в `readPump` или в `dispatch`.
- **Effort:** low

---

## [Качество кода и надежность]

### [62] Двойная блокировка: Room mutex внутри RoomManager mutex
- **Severity:** MEDIUM
- **Файл:** internal/server/room_manager.go:146-156 (UpdatePingTime)
- **Проблема:** `UpdatePingTime` вызывает `GetRoom` (берет RWMutex на чтение), затем `room.Lock()`. В `ws.go` handler-ы делают `manager.GetRoom()` → `room.Lock()` → операция → `room.Unlock()` → `manager.Broadcast()`. Broadcast берет свой RWMutex. Порядок блокировки: manager.mu → room.mu. Но в `collectGarbage` порядок: manager.mu (write lock) → чтение room.LastActivity (без room lock). Чтение `room.LastActivity` без room lock — data race.
- **Влияние:** В большинстве случаев безобидно (64-bit atomic read на x86/arm64), но формально это race condition. `go test -race` может не поймать, т.к. GC запускается по таймеру.
- **Рекомендация:** В `collectGarbage` брать `room.Lock()` перед чтением `room.LastActivity`, или сделать `LastActivity` атомарным (`atomic.Int64`).
- **Effort:** low

### [55] Отсутствие graceful drain для WebSocket при shutdown
- **Severity:** MEDIUM
- **Файл:** cmd/server/main.go:47-53
- **Проблема:** При shutdown вызывается `manager.CloseAll()`, который закрывает канал `done` всех клиентов. Но `conn.Close(websocket.StatusNormalClosure, "goodbye")` в `HandleWebSocket` может не успеть выполниться до `srv.Shutdown(ctx)`. `CloseAll` не ждет завершения горутин. Клиенты могут получить неожиданное закрытие соединения.
- **Влияние:** При рестарте пользователи могут увидеть ошибку вместо чистого disconnect. Reconnect-логика клиента сработает, но user experience будет хуже.
- **Рекомендация:** Отправлять WebSocket close frame (StatusGoingAway) из `CloseAll` и дать время на drain перед HTTP server shutdown.
- **Effort:** low

### [48] Send buffer drop без уведомления клиента
- **Severity:** MEDIUM
- **Файл:** internal/server/client.go:42-48
- **Проблема:** При переполнении send buffer (32 сообщения) — сообщение тихо отбрасывается с логом на сервере. Клиент не знает, что потерял сообщения. Его состояние расходится с серверным.
- **Влияние:** При медленном клиенте (плохая сеть) — UI показывает устаревшие данные. Голосования "пропадают", участники не появляются/исчезают.
- **Рекомендация:** При переполнении буфера — закрывать соединение с кодом ошибки, чтобы клиент переподключился и получил свежий room_state. Или отправлять room_state после detected overflow.
- **Effort:** low

### [45] Отсутствие ограничения на длину roomID
- **Severity:** MEDIUM
- **Файл:** internal/server/ws.go:27-31
- **Проблема:** `roomID` извлекается из URL без проверки длины. Можно создать комнату с roomID в мегабайты (через URL), что будет храниться в памяти как ключ map.
- **Влияние:** Memory exhaustion при создании большого количества комнат с длинными ID. Несмотря на rate limit создания (5/мин/IP), с множества IP можно забить память.
- **Рекомендация:** Ограничить длину roomID (например, 64 символа). Валидировать допустимые символы (alphanumeric + hyphen).
- **Effort:** low

---

## [Тестирование]

### [55] Отсутствие интеграционных WebSocket-тестов
- **Severity:** MEDIUM
- **Файл:** internal/server/ws.go (весь файл)
- **Проблема:** Файл ws.go (433 строки) содержит основную бизнес-логику обработки WS-сообщений (`handleJoin`, `handleVote`, `handleReveal` и т.д.) — ни один из этих handler-ов не покрыт тестами. Тесты есть только для HTTP handler-ов (handler_test.go), events, room_manager и domain. 
- **Влияние:** Ключевой flow (join → vote → reveal → new_round) не тестируется end-to-end. Регрессии в dispatch, join-логике, broadcast-логике — не ловятся.
- **Рекомендация:** Добавить integration-тесты с реальным WebSocket: `httptest.NewServer` + WebSocket client. Минимум: full join-vote-reveal flow, reconnect flow, error scenarios.
- **Effort:** medium

### [42] Отсутствие frontend-тестов
- **Severity:** MEDIUM
- **Файл:** web/ (вся директория)
- **Проблема:** Нет ни одного теста для frontend кода. state.ts, ws.ts, компоненты — всё без тестов. `npm test` не настроен.
- **Влияние:** Регрессии в клиентской логике (reconnect, state update, message handling) обнаруживаются только при ручном тестировании.
- **Рекомендация:** Добавить unit-тесты для ws.ts (mock WebSocket) и state.ts (signal updates). Vitest как test runner (уже есть Vite).
- **Effort:** medium

---

## [Наблюдаемость и операционная готовность]

### [68] Отсутствие structured logging
- **Severity:** MEDIUM
- **Файл:** все файлы с `log.Printf`
- **Проблема:** Используется `log.Printf` из stdlib — неструктурированные текстовые логи. Нет log levels (debug/info/warn/error). Нет request ID или connection ID для корреляции. Нет метрик.
- **Влияние:** При инцидентах — сложно фильтровать логи, отслеживать конкретное соединение, понимать масштаб проблемы. Нет мониторинга для alerting.
- **Рекомендация:** Заменить на `slog` (stdlib Go 1.21+, уже доступен при go 1.22). Добавить connectionID в контекст каждого WS-хэндлера. На будущее: prometheus metrics endpoint.
- **Effort:** medium

### [45] Health endpoint не защищен
- **Severity:** MEDIUM
- **Файл:** internal/server/handler.go:38-39
- **Проблема:** `/health` доступен публично без аутентификации. Отдает количество комнат, соединений, uptime. Это информация об инфраструктуре.
- **Влияние:** Атакующий может мониторить нагрузку сервера и выбирать оптимальный момент для атаки. Или определять что сервис живой.
- **Рекомендация:** Для простого проекта — приемлемо. При продакшен-использовании — вынести health на отдельный порт (admin port) или добавить basic auth.
- **Effort:** low

---

## [Сборка и деплой]

### [35] Docker FROM scratch — нет shell для отладки
- **Severity:** LOW
- **Файл:** Dockerfile:19-22
- **Проблема:** Runtime-образ `FROM scratch` не содержит ни shell, ни утилит. Невозможно `docker exec` для отладки в production. Нет CA-сертификатов (не критично для сервера, но может понадобиться для outbound HTTPS).
- **Влияние:** Сложнее отлаживать проблемы в runtime. Нет `curl`, `wget`, `ls`, даже `/bin/sh`.
- **Рекомендация:** Использовать `gcr.io/distroless/static:nonroot` или `alpine:3` для чуть большего размера (+5MB), но с минимальным tooling. Или оставить `scratch` и документировать ephemeral debug containers (`kubectl debug`).
- **Effort:** low

### [30] Go 1.22 в go.mod — потенциально устаревшая версия
- **Severity:** LOW
- **Файл:** go.mod:3
- **Проблема:** `go 1.22.0` в go.mod. На апрель 2026 текущая версия Go — 1.24+. Go 1.22 вышел из supported в ~2025. Dockerfile также использует `golang:1.22-alpine`.
- **Влияние:** Пропущены оптимизации, исправления безопасности в runtime. `slog` доступен с 1.21, enhanced routing в `net/http` с 1.22 (уже есть), но improved range over func, итераторы — с 1.23.
- **Рекомендация:** Обновить до Go 1.24. Протестировать с `-race`, проверить совместимость `nhooyr.io/websocket v1.8.17`.
- **Effort:** low

### [25] nhooyr.io/websocket v1.8.17 — unmaintained версия
- **Severity:** LOW
- **Файл:** go.mod:5
- **Проблема:** `nhooyr.io/websocket v1.8.17` — это последняя версия v1.x ветки, которая была переименована в `github.com/coder/websocket` (v2.x) в 2024. v1.x более не поддерживается автором.
- **Влияние:** Нет исправлений безопасности и багов для v1.x.
- **Рекомендация:** Мигрировать на `github.com/coder/websocket` v2.x. API совместим с минимальными изменениями.
- **Effort:** low

---

## [Архитектура frontend]

### [38] Отсутствие error boundary
- **Severity:** LOW
- **Файл:** web/src/app.tsx
- **Проблема:** Нет error boundary вокруг основного app tree. Необработанное исключение в любом компоненте — белый экран.
- **Влияние:** Пользователь теряет UI при runtime-ошибке. Нет способа восстановиться кроме reload.
- **Рекомендация:** Добавить Preact error boundary с fallback UI и кнопкой "Reload".
- **Effort:** low

### [35] Message queue растёт без ограничений при offline
- **Severity:** LOW
- **Файл:** web/src/ws.ts:19, 32-37
- **Проблема:** `messageQueue` — обычный массив без ограничения размера. При длительном offline и активных пользовательских действиях — очередь растет бесконечно. При reconnect — все сообщения отправляются залпом.
- **Влияние:** Потенциально: утечка памяти при длительном offline. При reconnect — burst сообщений, часть которых уже неактуальна (старые голоса, например).
- **Рекомендация:** Ограничить размер очереди (10-20). При reconnect — не flush старые сообщения, а отправить join (уже делается) и полагаться на room_state.
- **Effort:** low

### [30] Ручной роутинг без поддержки query parameters
- **Severity:** LOW
- **Файл:** web/src/app.tsx:17-26
- **Проблема:** Роутинг реализован через `currentPath` signal и `startsWith('/room/')`. Нет парсинга query string, нет поддержки 404 страницы (любой неизвестный path -> HomePage).
- **Влияние:** Минимальное для текущего scope (2 маршрута). Но расширяемость ограничена.
- **Рекомендация:** Для 2 маршрутов текущий подход — OK (KISS). При добавлении 3+ маршрутов — рассмотреть preact-router.
- **Effort:** low

---

## [Качество проекта]

### [22] LogMiddleware объявлен, но не используется
- **Severity:** LOW
- **Файл:** internal/server/handler.go:195-201
- **Проблема:** `LogMiddleware` экспортирован и имеет комментарий "unused but available". Мертвый код в production пакете.
- **Влияние:** Confusion для разработчиков. Нет request logging в production.
- **Рекомендация:** Либо подключить middleware в `NewServer`, либо удалить. Рекомендуется подключить — request logging полезен для отладки.
- **Effort:** low

### [18] CLAUDE.md указывает WS path `/ws/{roomId}`, что точнее реализации
- **Severity:** NOTE
- **Файл:** CLAUDE.md
- **Проблема:** CLAUDE.md корректно отражает реализацию (`/ws/room/{roomId}` — нет, написано `/ws/room/{roomId}` в описании, хотя endpoint: `GET /ws/room/{roomId}`). При этом в реализации — `/ws/{roomId}`. Рассинхрон трех источников: CLAUDE.md, docs/planning/04-architecture.md и код.
- **Влияние:** Путаница в документации.
- **Рекомендация:** Выбрать один canonical URL и обновить все три источника.
- **Effort:** low

### [15] Хорошая модульная структура domain/server
- **Severity:** NOTE
- **Файл:** internal/domain/, internal/server/
- **Проблема:** (Положительная находка) Разделение на `domain` (чистая бизнес-логика без I/O) и `server` (все I/O) — чистое, правильное. Domain не импортирует server. Зависимость односторонняя. Это хорошая архитектура для проекта такого масштаба.
- **Влияние:** Бизнес-логика тестируется изолированно (domain_test.go — 317 строк покрывают все scenarios). Легко менять transport без переписывания domain.
- **Рекомендация:** Сохранять текущий подход.
- **Effort:** —

### [12] Высокое соотношение тест/код для backend
- **Severity:** NOTE
- **Файл:** internal/
- **Проблема:** (Положительная находка) Backend: 1606 строк кода, 1659 строк тестов — соотношение ~1:1. Все тесты проходят, включая `-race`. Domain покрытие — отличное. Server package покрывает HTTP handlers, room manager, events, rate limiter.
- **Влияние:** Хорошая защита от регрессий.
- **Рекомендация:** Добавить WebSocket integration тесты (см. находку выше).
- **Effort:** —

### [10] Хорошая стратегия embedding с fallback
- **Severity:** NOTE
- **Файл:** internal/server/handler.go:73-113, web/embed.go
- **Проблема:** (Положительная находка) Трёхуровневый fallback для SPA: embedded FS → disk-based dist → placeholder HTML. Позволяет работать и как self-contained бинарник, и в dev-режиме с Vite, и без фронтенда вообще.
- **Влияние:** Отличный DX. Разработчик может запустить `go run ./cmd/server` без сборки фронтенда.
- **Рекомендация:** Сохранить текущий подход.
- **Effort:** —

### [8] Docker multi-stage build — хороший паттерн
- **Severity:** NOTE
- **Файл:** Dockerfile
- **Проблема:** (Положительная находка) 3-stage build: frontend (node:20-alpine) → backend (golang:1.22-alpine) → runtime (scratch). Минимальный размер image. CGO_ENABLED=0 для статической линковки.
- **Влияние:** Быстрый деплой, минимальная поверхность атаки.
- **Рекомендация:** Сохранить подход, обновить версии base images.
- **Effort:** —

---

## Саммари
- Всего находок: 22
- CRITICAL (90-100): 2
- HIGH (70-89): 3
- MEDIUM (40-69): 9
- LOW (20-39): 5
- NOTE (0-19): 5 (из них 4 положительные)
- Средний балл: 46.3
- Топ-3 проблемы:
  1. **[92]** Отсутствие авторизации на действия reveal/new_round/clear_room
  2. **[90]** WebSocket InsecureSkipVerify — отсутствие проверки Origin
  3. **[82]** SessionID генерируется на клиенте без серверной валидации

### Общая оценка

Проект демонстрирует зрелый подход к архитектуре: чистое разделение domain/server, минимальные зависимости (единственная — websocket library), хорошее тестовое покрытие backend. Стратегия embedding с 3-уровневым fallback — элегантна. Docker build — эффективный.

Основные риски лежат в области безопасности: отсутствие Origin проверки и авторизации делает сервис уязвимым для целенаправленных атак. Для внутрикомандного использования — приемлемо. Для публичного deployment — необходимо исправить CRITICAL находки.

Расхождения между архитектурной документацией и реализацией (WebSocket URL, roomName в join, message rate limit) требуют синхронизации — либо обновить docs, либо доработать код.
