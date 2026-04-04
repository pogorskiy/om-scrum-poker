# om-scrum-poker — Итоговый план реализации

**Дата:** 2026-04-04
**Статус:** Утверждён после 5 итераций планирования
**Вердикт:** ГОТОВ К РЕАЛИЗАЦИИ

---

## Оглавление

1. [Обзор проекта](#1-обзор-проекта)
2. [Технологический стек](#2-технологический-стек)
3. [Архитектура системы](#3-архитектура-системы)
4. [UX/UI дизайн](#4-uxui-дизайн)
5. [WebSocket протокол](#5-websocket-протокол)
6. [Бэкенд](#6-бэкенд)
7. [Фронтенд](#7-фронтенд)
8. [Безопасность](#8-безопасность)
9. [Деплой](#9-деплой)
10. [Документация для AI-агентов](#10-документация-для-ai-агентов)
11. [Оценка трудозатрат](#11-оценка-трудозатрат)
12. [Порядок реализации](#12-порядок-реализации)

---

## 1. Обзор проекта

**om-scrum-poker** — легковесный сервис для скрам-покера в реальном времени.

### Ключевые принципы
- **KISS** — минимум зависимостей, минимум абстракций, минимум кода
- **Один бинарник** — сервер + статика в одном файле
- **Без авторизации** — имя хранится в localStorage
- **Эфемерность** — комнаты живут в памяти, сервер stateless по дизайну
- **Мгновенный отклик** — WebSocket для всех действий в реальном времени

### Основные возможности
- Создание комнаты с уникальной ссылкой
- Присоединение по ссылке с вводом имени
- Голосование картами: `?`, `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100`
- Мгновенное раскрытие голосов с average, median
- Индикатор консенсуса и разброса оценок
- Система присутствия (зелёный/жёлтый/красный)
- Очистка голосов (новый раунд) и очистка комнаты

---

## 2. Технологический стек

### Бэкенд: Go (stdlib + 1 зависимость)
| Компонент | Решение | Обоснование |
|---|---|---|
| Язык | Go 1.22+ | Один статический бинарник, горутины для WebSocket, `embed.FS` для статики |
| HTTP | `net/http` (stdlib) | Достаточно для наших нужд, ноль зависимостей |
| WebSocket | `nhooyr.io/websocket` | Легковеснее gorilla, лучше API, поддержка контекстов |
| UUID | `crypto/rand` + `encoding/hex` | 4 строки кода вместо внешней библиотеки |
| **Итого зависимостей** | **1** | |

### Фронтенд: Preact + Signals
| Компонент | Решение | Обоснование |
|---|---|---|
| Фреймворк | Preact (~4 KB) | React API, но в 10 раз легче |
| Состояние | `@preact/signals` (~2 KB) | Тонкая реактивность без Redux/Zustand |
| CSS | Чистый CSS + BEM | Один файл на компонент, без препроцессоров |
| Сборка | Vite | Мгновенный HMR, esbuild под капотом |
| Язык | TypeScript (strict) | Типы для WebSocket контрактов |
| **Итого runtime зависимостей** | **2** | `preact` + `@preact/signals` |

### Почему именно этот стек
- **Go** — единственный язык, который даёт один статический бинарник без виртуальной машины, с нативным WebSocket и встраиванием статики
- **Preact** — минимальный React-совместимый фреймворк; Svelte или Solid тоже подойдут, но Preact проще для React-опытных разработчиков
- **Без Tailwind** — для 10 компонентов чистый CSS проще и не требует конфигурации
- **Без базы данных** — комнаты эфемерны, всё в памяти

---

## 3. Архитектура системы

### Высокоуровневая схема

```
┌─────────────────────────────────────────────┐
│                  Клиент (браузер)            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ HomePage │  │NameModal │  │ RoomPage │   │
│  └──────────┘  └──────────┘  └──────────┘   │
│         │              │           │         │
│         └──────────────┴───────────┘         │
│                    │                         │
│              [WebSocket]                     │
└────────────────────┼────────────────────────┘
                     │
┌────────────────────┼────────────────────────┐
│                Go-сервер                     │
│  ┌─────────────────┴──────────────────────┐  │
│  │          server (пакет)                │  │
│  │  ┌──────────┐  ┌──────────────┐        │  │
│  │  │HTTP Handler│ │WS Handler   │        │  │
│  │  └──────────┘  └──────────────┘        │  │
│  │  ┌──────────┐  ┌──────────────┐        │  │
│  │  │Rate Limit│  │Room Manager  │        │  │
│  │  └──────────┘  └──────────────┘        │  │
│  └─────────────────┬──────────────────────┘  │
│  ┌─────────────────┴──────────────────────┐  │
│  │          domain (пакет)                │  │
│  │  Room, Participant, Vote, Statistics   │  │
│  └────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────┐  │
│  │       embed.FS (статические файлы)     │  │
│  └────────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

### Два пакета (не четыре!)

Адвокат дьявола убедил команду, что 4-уровневая Clean Architecture — излишество для сервиса с одной сущностью и 6 операциями. Итого:

| Пакет | Содержимое | Правило |
|---|---|---|
| `internal/domain/` | `Room`, `Participant`, `VoteValue`, `RoundResult`, вычисление статистики | Чистая бизнес-логика. Ноль зависимостей. Ноль I/O. |
| `internal/server/` | HTTP/WS обработчики, Room Manager, Rate Limiter, GC, события | Вся инфраструктура и ввод/вывод |

### Конкурентность
- `sync.RWMutex` на карте комнат (Room Manager)
- `sync.Mutex` на каждой комнате
- Никакого I/O под блокировкой
- Рассылка через буферизированные каналы (32 сообщения)

---

## 4. UX/UI дизайн

### Экраны

#### 4.1 Главная страница
- Заголовок "om-scrum-poker" с подзаголовком "Simple. Self-hosted. No signup."
- Одно поле ввода: название комнаты (макс. 60 символов)
- Кнопка "Create Room" (активна только при непустом поле)
- Enter для отправки
- Переход на `/room/{slug}-{12 hex символов}`

#### 4.2 Модальное окно ввода имени
- Появляется если в localStorage нет имени
- "What should we call you?" + поле ввода (макс. 30 символов)
- Нельзя закрыть без ввода имени
- Имя сохраняется в localStorage, sessionId генерируется один раз

#### 4.3 Страница комнаты — фаза голосования
```
┌─────────────────────────────────────────────────┐
│  [om]  Sprint 42 Planning      [Copy Link] [⚙]  │
├─────────────────────────────────────────────────┤
│  Участники:                                      │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐            │
│  │🟢 Ana│ │🟢 Bob│ │🟡 Cat│ │🔴 Dan│            │
│  │  [✓] │ │  [ ] │ │  [✓] │ │  [-] │            │
│  └──────┘ └──────┘ └──────┘ └──────┘            │
├─────────────────────────────────────────────────┤
│       [ Show Votes (2 of 4 voted) ]              │
├─────────────────────────────────────────────────┤
│  [?] [0] [0.5] [1] [2] [3] [5] [8] [13]...     │
├─────────────────────────────────────────────────┤
│  [New Round]                    [Clear Room]     │
└─────────────────────────────────────────────────┘
```

#### 4.4 Страница комнаты — фаза раскрытия
- Карты переворачиваются, показывая значения
- Статистика: Average, Median, число неопределившихся ("?")
- Индикатор консенсуса (зелёный) или разброса (информационный)
- Карточная колода заблокирована до нового раунда

### Система присутствия

| Состояние | Цвет | Условие |
|---|---|---|
| Active (онлайн) | 🟢 Зелёный `#22c55e` | Вкладка в фокусе или активность < 2 мин назад |
| Idle (отошёл) | 🟡 Жёлтый `#eab308` | Вкладка скрыта или нет активности > 2 мин |
| Disconnected (отключён) | 🔴 Красный `#ef4444` | WebSocket закрыт, 2 пропущенных пинга (> 10 сек) |

- Отключённые пользователи остаются в списке (приглушены, 60% opacity)
- Их голоса сохраняются при раскрытии
- Удаляются только при "Clear Room" или GC (24 часа)

### Адаптивный дизайн

| Брейкпоинт | Поведение |
|---|---|
| < 640px (мобильный) | Карты в 2 ряда по 6, участники в 2 колонки, минимум 44px touch target |
| 640-1024px (планшет) | Свободная обтекаемость карт участников |
| > 1024px (десктоп) | Всё на одном экране без прокрутки |

### Тосты
- Позиция: top-center
- Длительность: 2 секунды + 300ms fade
- Максимум 3 видимых
- Примеры: "Link copied!", "Ana joined", "New round started"

### Подтверждение
- Только для "Clear Room" (деструктивное действие)
- Модальное окно: "Clear Room? This will remove all participants."

---

## 5. WebSocket протокол

> **Единый источник правды** — Architecture Doc v3.1, Section 4.
> Фронтенд и бэкенд планы ссылаются на архитектурный документ, не переопределяют протокол.

### Формат конверта
```json
{ "type": "event_name", "payload": { ... } }
```

### Все события (19 штук)

#### Клиент → Сервер (8 событий)

| Событие | Payload | Описание |
|---|---|---|
| `join` | `{ sessionId, userName, roomName }` | Присоединение к комнате |
| `vote` | `{ value }` | Голос (или `value: ""` для отмены) |
| `reveal` | `{}` | Раскрыть голоса |
| `new_round` | `{}` | Новый раунд |
| `clear_room` | `{}` | Очистить комнату |
| `update_name` | `{ userName }` | Изменить имя |
| `presence` | `{ status }` | Обновить статус ("active" / "idle") |
| `leave` | `{}` | Явный выход из комнаты |

#### Сервер → Клиенты (11 событий)

| Событие | Payload | Описание |
|---|---|---|
| `room_state` | `{ roomId, roomName, phase, participants, result }` | Полное состояние при подключении |
| `participant_joined` | `{ sessionId, userName, status }` | Кто-то присоединился |
| `participant_left` | `{ sessionId }` | Кто-то вышел |
| `vote_cast` | `{ sessionId }` | Кто-то проголосовал (без значения) |
| `vote_retracted` | `{ sessionId }` | Кто-то отменил голос |
| `votes_revealed` | `{ votes, average, median, uncertainCount, totalVoters, hasConsensus, spread }` | Результаты голосования |
| `round_reset` | `{}` | Раунд сброшен |
| `room_cleared` | `{}` | Комната очищена |
| `presence_changed` | `{ sessionId, status }` | Статус участника изменился |
| `name_updated` | `{ sessionId, userName }` | Имя участника изменилось |
| `error` | `{ code, message }` | Ошибка |

### Коды ошибок
- `room_not_found` — комната не найдена
- `invalid_name` — невалидное имя
- `invalid_vote` — невалидный голос
- `rate_limited` — превышен лимит
- `room_full` — комната полна (если когда-то добавим лимит)

### Heartbeat
- **Только на уровне протокола WebSocket** (ping/pong)
- Сервер пингует каждые 5 секунд
- Если 2 пинга пропущено (10 сек) → статус "disconnected"
- Никаких application-level heartbeat сообщений

### Переподключение (клиент)
- Exponential backoff: 500ms → 10s (cap), джиттер 30%
- После 30 секунд неудач: баннер "Connection lost. [Retry]"
- При переподключении: повторная отправка `join` → сервер восстанавливает состояние

---

## 6. Бэкенд

### Структура проекта
```
cmd/
  server/
    main.go              # Точка входа, конфигурация, graceful shutdown
internal/
  domain/
    room.go              # Room, Participant, VoteValue, Phase
    room_test.go         # Юнит-тесты бизнес-логики
    stats.go             # CalculateResult (average, median, consensus)
    stats_test.go        # Табличные тесты для статистики
  server/
    handler.go           # HTTP обработчики (health, room check, SPA fallback)
    ws.go                # WebSocket upgrade, read/write циклы
    client.go            # Структура Client (conn, send channel, session)
    room_manager.go      # In-memory хранилище комнат, CRUD, GC
    ratelimit.go         # Token bucket per IP
    events.go            # Типы сообщений, сериализация
```

### Доменная модель

```go
type Phase string
const (
    PhaseVoting Phase = "voting"
    PhaseReveal Phase = "reveal"
)

type VoteValue string  // "", "?", "0", "0.5", "1", "2", "3", "5", "8", "13", "20", "40", "100"

type Participant struct {
    SessionID   string
    Name        string
    Vote        VoteValue
    Status      string     // "active", "idle", "disconnected"
    LastPing    time.Time
}

type Room struct {
    ID           string
    Name         string
    Phase        Phase
    Participants map[string]*Participant
    CreatedAt    time.Time
    LastActivity time.Time
    mu           sync.Mutex
}

type RoundResult struct {
    Votes          []VoteEntry
    Average        *float64  // nil если нет числовых голосов
    Median         *float64
    UncertainCount int
    TotalVoters    int
    HasConsensus   bool
    Spread         *[2]float64  // [min, max] или nil
}
```

### Room ID
- Формат: `{slug}-{12 hex символов}` (пример: `sprint-42-a3f1c9b2e4d6`)
- 48 бит энтропии (~281 триллион комбинаций)
- Генерируется на клиенте при создании комнаты
- Клиент генерирует URL и переходит → при первом `join` сервер создаёт комнату

### Rate Limiting
- Token bucket per IP
- Создание комнат (через WS join): 5 в минуту
- WebSocket соединения: 20 в минуту
- Ответ: error событие с кодом `rate_limited`

### Garbage Collection
- Фоновая горутина каждые 10 минут
- Удаляет комнаты без подключений и `lastActivity > 24 часа`

### Конфигурация (3 переменных окружения)
| Переменная | По умолчанию | Описание |
|---|---|---|
| `PORT` | `8080` | Порт сервера |
| `HOST` | `0.0.0.0` | Адрес привязки |
| `TRUST_PROXY` | `false` | Доверять X-Forwarded-For |

Все остальные значения — захардкоженные константы (TTL, буферы, лимиты).

---

## 7. Фронтенд

### Структура проекта
```
web/
  index.html
  vite.config.ts
  tsconfig.json
  src/
    main.tsx                    # Точка входа
    app.tsx                     # Роутер (2 маршрута)
    state.ts                    # Все сигналы, типы, localStorage
    ws.ts                       # WebSocket клиент, обработчики
    tokens.css                  # CSS custom properties (палитра)
    global.css                  # Сброс, типографика
    hooks/
      usePresence.ts            # Отслеживание active/idle
    utils/
      room-url.ts               # Генерация и парсинг URL комнат
    components/
      HomePage/
        HomePage.tsx
        HomePage.css
      NameEntryModal/
        NameEntryModal.tsx
        NameEntryModal.css
      RoomPage/
        RoomPage.tsx
        RoomPage.css
      Header/
        Header.tsx              # + Copy Link inline
        Header.css
      ParticipantList/
        ParticipantList.tsx
        ParticipantList.css
      ParticipantCard/
        ParticipantCard.tsx
        ParticipantCard.css
      CardDeck/
        CardDeck.tsx
        CardDeck.css
      Card/
        Card.tsx
        Card.css
      Toast/
        Toast.tsx
        Toast.css
      ConfirmDialog/
        ConfirmDialog.tsx
        ConfirmDialog.css
```

### 10 компонентов (не 16!)

Адвокат дьявола сократил: CopyLinkButton, ReconnectionBanner, ActionButtons, SettingsDropdown, Statistics — всё заинлайнено в родительские компоненты.

### Управление состоянием
- **Один файл `state.ts`** с Preact Signals
- Сигналы: `roomState`, `connectionStatus`, `userName`, `sessionId`, `selectedCard`, `toasts`
- Computed: `isRevealed`, `myParticipant`, `voteCount`
- localStorage: `userName` и `sessionId` (с try/catch для приватного режима)

### Роутинг
- Кастомный роутер (~20 строк), НЕ библиотека
- Два маршрута: `/` → HomePage, `/room/:id` → RoomPage
- `popstate` для навигации назад/вперёд

### Статистика
- **Сервер вычисляет всё** — average, median, consensus, spread
- Фронтенд только отображает данные из `votes_revealed`
- Нет `utils/stats.ts` — убрали как дублирование

### Оценка бандла
- ~16-21 KB gzipped (Preact + Signals + наш код)

---

## 8. Безопасность

### Что защищаем (простые атаки)

| Атака | Защита |
|---|---|
| Спам создания комнат | Rate limit: 5 комнат/мин per IP |
| Перебор room ID | 48 бит энтропии (281 триллион), rate limit на WS подключения |
| XSS через имя пользователя | Preact экранирует по умолчанию, валидация на сервере |
| Огромные сообщения | Максимум 1 KB на WebSocket сообщение |
| Переполнение соединений | Лимит WS подключений per IP |
| CORS | Стандартная политика (статика и WS с одного origin) |

### Что НЕ защищаем (осознанно)
- Утечка имён пользователей — принято как допустимое
- Подмена identity — нет авторизации, sessionId в localStorage
- DDoS — требует внешнюю инфраструктуру (CloudFlare и т.д.)

---

## 9. Деплой

### Один бинарник
```bash
# Сборка
cd web && npm run build
cd .. && go build -o om-scrum-poker ./cmd/server

# Запуск
PORT=3000 ./om-scrum-poker
```

Фронтенд встраивается в Go-бинарник через `go:embed`.

### Docker
```dockerfile
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/ .
RUN npm ci && npm run build

FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist web/dist
RUN CGO_ENABLED=0 go build -o /om-scrum-poker ./cmd/server

FROM scratch
COPY --from=backend /om-scrum-poker /om-scrum-poker
EXPOSE 8080
ENTRYPOINT ["/om-scrum-poker"]
```

Размер образа: ~15 MB.

### Health Check
```
GET /health → { "status": "ok", "rooms": 5, "connections": 23, "uptime": "2h15m" }
```

---

## 10. Документация для AI-агентов

### Стратегия: 2 файла, не больше

| Файл | Содержимое |
|---|---|
| `CLAUDE.md` | Команды сборки, структура проекта, архитектурные решения, протокол (ссылка на arch doc) |
| `docs/planning/04-architecture.md` | Полный архитектурный документ — единый источник правды для протокола |

### Принципы AI-friendly кода
- Короткие файлы (< 200 строк)
- Говорящие имена функций и переменных
- Типы TypeScript для всех WebSocket сообщений
- Табличные тесты в Go для лёгкого понимания бизнес-правил
- Комментарии только где логика неочевидна

---

## 11. Оценка трудозатрат

### Объём кода

| Компонент | Строки |
|---|---|
| Бэкенд Go (production) | ~1,000 |
| Бэкенд Go (тесты) | ~450 |
| Фронтенд TypeScript | ~840 |
| Фронтенд CSS | ~450 |
| Инфраструктура (Docker, Makefile, CLAUDE.md) | ~80 |
| **Итого** | **~2,850** |

### Время реализации

| Фаза | Оценка |
|---|---|
| Бэкенд: domain + тесты | 3-4 часа |
| Бэкенд: server (WS, HTTP, room manager, GC) | 6-8 часов |
| Бэкенд: интеграционные тесты | 2-3 часа |
| Фронтенд: компоненты + состояние + роутинг | 6-8 часов |
| Фронтенд: CSS (responsive + все состояния) | 3-4 часа |
| Фронтенд: accessibility | 1-2 часа |
| Интеграция (embed, Docker, Makefile) | 1-2 часа |
| Тестирование и полировка | 2-3 часа |
| **Итого** | **24-34 часа (~4-5 рабочих дней)** |

---

## 12. Порядок реализации

### Фаза 1: Бэкенд Domain (3-4 часа)
1. `internal/domain/room.go` — Room, Participant, VoteValue, Phase, методы
2. `internal/domain/stats.go` — CalculateResult (average, median, consensus, spread)
3. `internal/domain/room_test.go` + `stats_test.go` — полное покрытие табличными тестами

### Фаза 2: Бэкенд Server (8-11 часов)
4. `internal/server/events.go` — типы сообщений, сериализация
5. `internal/server/client.go` — структура клиента
6. `internal/server/room_manager.go` — хранилище комнат + GC
7. `internal/server/ws.go` — WebSocket handler, read/write, dispatch
8. `internal/server/handler.go` — HTTP: health, room check, SPA fallback
9. `internal/server/ratelimit.go` — token bucket per IP
10. `cmd/server/main.go` — точка входа, graceful shutdown

### Фаза 3: Фронтенд (10-14 часов)
11. Инициализация проекта: `vite.config.ts`, `tsconfig.json`, `index.html`
12. `src/state.ts` — все сигналы, типы, localStorage
13. `src/ws.ts` — WebSocket клиент с reconnect
14. `src/app.tsx` + `src/main.tsx` — роутер, точка входа
15. `src/hooks/usePresence.ts` + `src/utils/room-url.ts`
16. Компоненты: HomePage → NameEntryModal → Header → CardDeck + Card → ParticipantList + ParticipantCard → RoomPage → Toast → ConfirmDialog
17. CSS: tokens.css, global.css, компонентные стили
18. Responsive и accessibility

### Фаза 4: Интеграция (3-5 часов)
19. `go:embed` для встраивания статики
20. Dockerfile
21. Makefile (build, dev, test)
22. CLAUDE.md
23. Ручное тестирование в нескольких браузерах

---

## Приложение: Результаты итеративного планирования

Планирование прошло 5 итераций с участием 6 ролей:

| Итерация | Что произошло |
|---|---|
| **1** | Продуктовый дизайнер создал UX-спецификацию (35 требований) |
| **1** | Frontend Lead, Backend Lead, System Architect создали начальные планы параллельно |
| **1 ревью** | PO/Architect + Devil's Advocate нашли 25 проблем: 6 противоречий протокола, 4-уровневая архитектура — излишество, 16 компонентов → 10 |
| **2** | Все три лидера обновили планы: 2 пакета вместо 4, 10 компонентов, 1 зависимость Go |
| **2 ревью** | 18 проблем: протокол определён в 3 местах и расходится, формат конверта противоречив |
| **3** | Архитектор создал единый канонический протокол. Frontend и Backend выровнялись |
| **3 ревью** | 1 баг: `"revealed"` vs `"reveal"`, 2 мелких несоответствия |
| **4** | Хирургические правки всех 3 документов |
| **5 (финал)** | 19/19 событий совпадают, 35/35 UX-требований покрыты, 0 противоречий. **APPROVED.** |

---

*Детальные технические документы (на английском):*
- [01-ux-design.md](01-ux-design.md) — UX/UI спецификация
- [02-frontend-plan.md](02-frontend-plan.md) — Frontend план v3.1
- [03-backend-plan.md](03-backend-plan.md) — Backend план v3.1
- [04-architecture.md](04-architecture.md) — Архитектура v3.1 (единый источник правды)
