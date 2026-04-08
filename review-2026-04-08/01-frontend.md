# Ведущий фронтенд-разработчик — Ревью проекта
- Дата: 2026-04-08
- Статус: done
- Охват: web/package.json, web/tsconfig.json, web/vite.config.ts, web/vitest.config.ts, web/index.html, web/src/main.tsx, web/src/app.tsx, web/src/state.ts, web/src/ws.ts, web/src/ws.test.ts, web/src/observer.test.ts, web/src/design-tokens.test.ts, web/src/tokens.css, web/src/global.css, web/src/test-setup.ts, web/src/vite-env.d.ts, web/src/utils/room-url.ts, web/src/hooks/usePresence.ts, web/src/components/HomePage/HomePage.tsx, web/src/components/HomePage/HomePage.css, web/src/components/RoomPage/RoomPage.tsx, web/src/components/RoomPage/RoomPage.css, web/src/components/RoomPage/RoomPage.test.tsx, web/src/components/Header/Header.tsx, web/src/components/Header/Header.css, web/src/components/Card/Card.tsx, web/src/components/Card/Card.css, web/src/components/Card/Card.test.tsx, web/src/components/CardDeck/CardDeck.tsx, web/src/components/CardDeck/CardDeck.css, web/src/components/CardDeck/CardDeck.test.tsx, web/src/components/ParticipantList/ParticipantList.tsx, web/src/components/ParticipantList/ParticipantList.css, web/src/components/ParticipantCard/ParticipantCard.tsx, web/src/components/ParticipantCard/ParticipantCard.css, web/src/components/ParticipantCard/ParticipantCard.test.tsx, web/src/components/NameEntryModal/NameEntryModal.tsx, web/src/components/NameEntryModal/NameEntryModal.css, web/src/components/NameEntryModal/NameEntryModal.test.tsx, web/src/components/EditNameModal/EditNameModal.tsx, web/src/components/EditNameModal/EditNameModal.css, web/src/components/EditNameModal/EditNameModal.test.tsx, web/src/components/Modal/Modal.tsx, web/src/components/Modal/Modal.css, web/src/components/Modal/Modal.test.tsx, web/src/components/ConfirmDialog/ConfirmDialog.tsx, web/src/components/ConfirmDialog/ConfirmDialog.css, web/src/components/ConfirmDialog/ConfirmDialog.test.tsx, web/src/components/ConnectionBanner/ConnectionBanner.tsx, web/src/components/ConnectionBanner/ConnectionBanner.css, web/src/components/ConnectionBanner/ConnectionBanner.test.tsx, web/src/components/Timer/Timer.tsx, web/src/components/Timer/Timer.css, web/src/components/Timer/Timer.test.tsx, web/src/components/Toast/Toast.tsx, web/src/components/Toast/Toast.css, web/src/components/Footer/Footer.tsx, web/src/components/Footer/Footer.css, web/src/components/Footer/Footer.test.tsx
- Модель: Claude Opus 4.6

---

## [Протокол / Совместимость с бэкендом]

### [75] Протокол WebSocket не документирован для timer и role событий
- **Severity:** HIGH
- **Файл:** web/src/state.ts:63-76, docs/planning/04-architecture.md:387-411
- **Проблема:** Фронтенд определяет и использует 6 типов сообщений (`update_role`, `timer_set_duration`, `timer_start`, `timer_reset`, `timer_updated`, `role_updated`), которые полностью отсутствуют в каноническом протоколе (Section 4 architecture doc). Таблица событий в спецификации заявлена как исчерпывающая: "This table lists ALL WebSocket events. There are no other events." Бэкенд реально поддерживает эти события (проверено в `internal/server/events.go`), но спецификация не обновлена.
- **Влияние:** Любой новый разработчик или агент, опирающийся на спецификацию, создаст несовместимую реализацию. Нарушен принцип "SINGLE SOURCE OF TRUTH" заявленный в самой спецификации. При рефакторинге бэкенда эти события могут быть случайно удалены.
- **Рекомендация:** Обновить `docs/planning/04-architecture.md` Section 4.3, добавив все 6 событий с полными payload-примерами. Обновить CLAUDE.md (раздел WebSocket Protocol).
- **Effort:** low

### [62] Тип `join` payload включает `role`, но спецификация этого не указывает
- **Severity:** MEDIUM
- **Файл:** web/src/state.ts:65, web/src/ws.ts:346-348
- **Проблема:** Фронтенд отправляет `role` в payload `join` сообщения (`{ sessionId, userName, roomName, role }`), но каноническая спецификация определяет payload `join` как содержащий только `sessionId`, `userName`, `roomName`. Бэкенд принимает `role` (проверено: `ParticipantJoinedPayload` содержит поле `Role`), но это не документировано.
- **Влияние:** Рассинхронизация спецификации и реализации. Меньший риск чем для timer/role событий, т.к. поле опционально.
- **Рекомендация:** Добавить поле `role` (опциональное, default `"voter"`) в спецификацию `join` payload.
- **Effort:** low

---

## [Состояние / Управление данными]

### [72] Очередь сообщений растёт без ограничений при отключении WebSocket
- **Severity:** HIGH
- **Файл:** web/src/ws.ts:45-51
- **Проблема:** Функция `send()` при отсутствии соединения складывает сообщения в `messageQueue` без лимита. Если пользователь активно кликает карточки или взаимодействует с UI при потере соединения, очередь будет расти неограниченно. При reconnect все накопленные сообщения отправляются залпом (`flushQueue`), что может перегрузить сервер и отправить устаревшие данные (например, старые голоса).
- **Влияние:** Утечка памяти при длительном отключении с активным UI. Некорректное состояние после reconnect (устаревшие vote, presence сообщения). Спецификация бэкенда указывает send buffer 32 сообщения, а фронтенд не ограничивает.
- **Рекомендация:** 1) Ограничить очередь до 32 сообщений (как в бэкенде). 2) При reconnect отправлять только `join`, без flush устаревших команд, т.к. `room_state` от сервера всё равно восстановит состояние.
- **Effort:** low

### [58] Timer mutates сигнал напрямую из компонента
- **Severity:** MEDIUM
- **Файл:** web/src/components/Timer/Timer.tsx:29
- **Проблема:** Компонент `Timer` напрямую мутирует глобальный сигнал `timerState` при достижении нуля: `timerState.value = { ...timerState.value, state: 'expired', remaining: 0 }`. Это нарушает однонаправленный поток данных: состояние таймера должно обновляться только через WebSocket-обработчик (`handleMessage`), а не из UI-компонента. Кроме того, если сервер отправит `timer_updated` с отличающимся состоянием, возникнет гонка.
- **Влияние:** Возможная рассинхронизация клиентского и серверного состояния таймера. Нарушение архитектурной конвенции (сигналы обновляются из ws.ts).
- **Рекомендация:** Использовать локальный state для отображения (`displayRemaining`), а expired-состояние определять через `displayRemaining <= 0` без мутации глобального сигнала. Либо вынести логику автоматического expire в ws.ts.
- **Effort:** medium

### [52] Отсутствует валидация `RoomState` при получении с сервера
- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:69-75
- **Проблема:** `isValidServerMessage` проверяет только наличие `type` (string) и `payload` (exists). Не проверяется структура payload для каждого типа сообщения. Если сервер отправит `room_state` с некорректной структурой (отсутствующие поля `participants`, `phase`), код упадет при обращении к `.participants.map()` и т.п. Это особенно важно при обновлении бэкенда.
- **Влияние:** Runtime ошибки при несовместимых изменениях бэкенда. Сложная отладка (ошибка проявится глубоко в компоненте, а не на уровне получения сообщения).
- **Рекомендация:** Добавить хотя бы базовую проверку критических полей для `room_state` (наличие `participants` как массив, `phase` как string). Не обязательно полная JSON Schema — достаточно guard-условий.
- **Effort:** medium

---

## [UI/UX]

### [65] Нет защиты от XSS в отображаемых именах пользователей
- **Severity:** MEDIUM
- **Файл:** web/src/components/ParticipantCard/ParticipantCard.tsx:70, web/src/components/Header/Header.tsx:45
- **Проблема:** Имена пользователей приходящие с сервера (через `participant_joined`, `name_updated`, `room_state`) отображаются напрямую через `{participant.userName}` и `{userName.value}`. Хотя Preact/JSX по умолчанию экранирует HTML-сущности (что предотвращает прямую HTML-инъекцию), имя может содержать управляющие Unicode-символы, RTL override, zero-width joiners и подобные символы, способные нарушить layout. Бэкенд выполняет sanitization (zalgo, emoji), но фронтенд полностью доверяет бэкенду.
- **Влияние:** При обходе серверной валидации (или при подключении к другому серверу) возможно нарушение отображения. На практике риск минимален благодаря серверной защите.
- **Рекомендация:** Добавить клиентскую sanitization как defense-in-depth: ограничить длину отображаемого имени, обрезать управляющие символы. Это не требует дублирования серверной логики — достаточно базового `displayName.slice(0, 30)` и фильтрации control characters.
- **Effort:** low

### [55] `ConfirmDialog` использует хардкодированные id для aria-атрибутов
- **Severity:** MEDIUM
- **Файл:** web/src/components/ConfirmDialog/ConfirmDialog.tsx:13
- **Проблема:** `ariaLabelledBy="confirm-title"` и `ariaDescribedBy="confirm-message"` — фиксированные id. Если на странице окажутся два `ConfirmDialog` одновременно (маловероятно, но возможно), id будут дублироваться, что нарушает валидность HTML и работу screen readers.
- **Влияние:** Нарушение accessibility при множественных диалогах. На текущий момент используется только один `ConfirmDialog` (Clear Room), но это хрупкая неявная зависимость.
- **Рекомендация:** Генерировать уникальные id с помощью `useId()` (Preact 10.19+) или передавать prefix через props.
- **Effort:** low

### [48] Отсутствует индикация загрузки при первом подключении к комнате
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:62-64
- **Проблема:** При статусе `connecting` отображается простой текст "Connecting..." без анимации или визуального индикатора. Пользователь не получает feedback о прогрессе. Более того, при медленном соединении нет timeout-сообщения и нет кнопки отмены.
- **Влияние:** Плохой UX при медленном соединении. Пользователь не понимает, происходит ли что-то.
- **Рекомендация:** Добавить спиннер (аналогичный ConnectionBanner) и текст "Connecting to room...". При длительном ожидании (>5с) показать дополнительное сообщение.
- **Effort:** low

### [45] Кнопки в header не имеют aria-label для иконочных кнопок
- **Severity:** MEDIUM
- **Файл:** web/src/components/Header/Header.tsx:56-69
- **Проблема:** Кнопка "Copy Link" имеет текст, но кнопка очистки комнаты (`header__clear-btn`) имеет только `title="Clear room"` и SVG-иконку без текста. Атрибут `title` не считается достаточным для accessibility — screen readers могут его не озвучить. Нужен `aria-label`.
- **Влияние:** Пользователи screen readers не поймут назначение кнопки.
- **Рекомендация:** Добавить `aria-label="Clear room"` к кнопке очистки. Аналогично для кнопки редактирования имени (edit-btn) добавить `aria-label="Change name"`.
- **Effort:** low

### [42] Нет обработки ситуации когда `roomId` содержит спецсимволы
- **Severity:** MEDIUM
- **Файл:** web/src/utils/room-url.ts:29-32
- **Проблема:** `parseRoomId` принимает только `[a-z0-9-]+`, что корректно для валидных room ID. Однако `generateRoomUrl` при пустом `name` (после slugify) генерирует URL только из hex-символов, что работает. Потенциальная проблема: пользователь может вручную ввести URL с невалидным room ID (заглавные буквы, подчёркивания) — `parseRoomId` вернёт `null`, и пользователь увидит "Invalid room URL" без объяснения.
- **Влияние:** Плохой UX при ручном вводе URL. Минимальный риск т.к. ссылки обычно копируются.
- **Рекомендация:** Показать более информативное сообщение об ошибке или попытаться нормализовать URL (toLower, замена `_` на `-`).
- **Effort:** low

---

## [CSS / Визуальные проблемы]

### [60] Input-элементы не имеют фона — проблемы в dark mode
- **Severity:** MEDIUM
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.css:17-24, web/src/components/EditNameModal/EditNameModal.css:15-22, web/src/components/HomePage/HomePage.css:39-48
- **Проблема:** Ни один input-элемент не имеет явно заданного `background-color`. В `global.css` для `input` задан `border: none` и `font-family: inherit`, но фон не сброшен. Большинство браузеров по умолчанию задают белый фон для input. В dark mode (`prefers-color-scheme: dark`) это приводит к белому полю ввода на тёмном фоне формы, создавая резкий визуальный контраст и нарушая цветовую схему.
- **Влияние:** Визуальный баг в dark mode для всех пользователей с системной тёмной темой. Текст в input может стать нечитаемым (тёмный текст на светлом фоне при тёмном окружении).
- **Рекомендация:** Добавить `background-color: var(--color-surface)` и `color: var(--color-text)` в базовый стиль input в `global.css` или в каждый компонентный CSS.
- **Effort:** low

### [38] ConnectionBanner.css использует rem вместо дизайн-токенов
- **Severity:** LOW
- **Файл:** web/src/components/ConnectionBanner/ConnectionBanner.css:25-27, 62-63
- **Проблема:** `ConnectionBanner.css` использует литеральные значения `0.5rem`, `1rem`, `0.875rem`, `0.25rem`, `0.75rem` вместо CSS-переменных `--space-*`. Все остальные компоненты консистентно используют токены. Это единственный CSS-файл с таким нарушением.
- **Влияние:** Нарушение консистентности. При изменении дизайн-токенов этот компонент не обновится автоматически.
- **Рекомендация:** Заменить литеральные rem-значения на соответствующие `--space-*` токены: `0.5rem` -> `var(--space-sm)`, `1rem` -> `var(--space-md)`, `0.25rem` -> `var(--space-xs)` и т.д.
- **Effort:** low

### [35] ConfirmDialog использует `--color-status-disconnected` вместо `--color-danger` для кнопки подтверждения
- **Severity:** LOW
- **Файл:** web/src/components/ConfirmDialog/ConfirmDialog.css:45-46
- **Проблема:** `.confirm-dialog__btn--confirm` использует `background: var(--color-status-disconnected)` — токен предназначенный для индикации статуса отключения. Семантически правильный токен для деструктивного действия — `--color-danger`. В hover-состоянии уже используется `--color-danger-hover`, создавая несогласованность.
- **Влияние:** Семантическое несоответствие. При изменении цвета статуса disconnected неожиданно изменится цвет кнопки подтверждения.
- **Рекомендация:** Заменить `var(--color-status-disconnected)` на `var(--color-danger)`.
- **Effort:** low

---

## [Архитектура / Качество кода]

### [50] Отсутствие тестов для Header компонента
- **Severity:** MEDIUM
- **Файл:** web/src/components/Header/Header.tsx
- **Проблема:** Компонент `Header` — один из самых сложных компонентов (содержит Copy Link, Edit Name, Clear Room, Role Toggle), но не имеет ни одного теста. Это единственный компонент с бизнес-логикой без тестового файла.
- **Влияние:** Регрессии в Header не будут обнаружены автоматически. Copy Link (clipboard API), role toggle, clear room — всё нетривиальная логика.
- **Рекомендация:** Создать `Header.test.tsx` с тестами: рендер room name, clipboard copy (мок navigator.clipboard), role toggle (send update_role), clear room confirm flow.
- **Effort:** medium

### [47] Отсутствие тестов для HomePage компонента
- **Severity:** MEDIUM
- **Файл:** web/src/components/HomePage/HomePage.tsx
- **Проблема:** Компонент `HomePage` содержит form submission и навигацию, но не имеет тестов.
- **Влияние:** Регрессии в создании комнаты не обнаружатся автоматически.
- **Рекомендация:** Создать `HomePage.test.tsx` с тестами: рендер формы, disabled-состояние кнопки, submit с навигацией, trim пустого имени.
- **Effort:** low

### [44] Отсутствие тестов для Toast компонента
- **Severity:** MEDIUM
- **Файл:** web/src/components/Toast/Toast.tsx
- **Проблема:** `Toast` компонент не имеет тестов. Хотя компонент простой, он содержит условную логику (role/aria-live атрибуты зависят от типа).
- **Влияние:** Изменения в toast-логике могут нарушить accessibility-атрибуты незамеченно.
- **Рекомендация:** Создать `Toast.test.tsx`: рендер пустого состояния, рендер info/error toast, проверка role/aria-live атрибутов.
- **Effort:** low

### [43] Отсутствие тестов для ParticipantList компонента
- **Severity:** MEDIUM
- **Файл:** web/src/components/ParticipantList/ParticipantList.tsx
- **Проблема:** `ParticipantList` не имеет тестов. Хотя компонент тривиальный (map по participants), он является критической точкой — UX-правило запрещает пересортировку по статусу, и это правило стоит закрепить тестом.
- **Влияние:** Если кто-то добавит сортировку по статусу, это нарушит UX-правило и не будет обнаружено тестами.
- **Рекомендация:** Создать `ParticipantList.test.tsx` с тестом: порядок участников соответствует порядку от сервера (join order), не пересортирован по статусу.
- **Effort:** low

### [40] `isFirstRoomState` — хрупкий модульный стейт без тестов
- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:31, 108-111
- **Проблема:** Переменная `isFirstRoomState` используется для показа toast при первом подключении к пустой комнате. Она сбрасывается в `connect()` (строка 328), но не сбрасывается в `disconnect()`. Если пользователь быстро переключается между комнатами, переменная может быть в неожиданном состоянии. Логика "показать hint при пустой комнате" зависит от race condition между reconnect и room_state.
- **Влияние:** Toast может показаться при reconnect к существующей комнате (если все остальные участники уже ушли). Минорное UX-замечание.
- **Рекомендация:** Задокументировать поведение. Рассмотреть сброс `isFirstRoomState` в `disconnect()`. Добавить тест.
- **Effort:** low

---

## [Типизация]

### [46] Неполное типирование `ServerMessage` — отсутствует `Vote` в `ParticipantInfo`
- **Severity:** MEDIUM
- **Файл:** web/src/state.ts:50, web/src/ws.ts:97-101
- **Проблема:** Тип `ServerMessage` для `participant_joined` определяет `role` как `string` с optional (`role?: string`), тогда как бэкенд всегда отправляет `role` (поле обязательное в `ParticipantJoinedPayload`). Также `status` типизирован как `string` вместо union `'active' | 'idle' | 'disconnected'`, и `role` как `string` вместо `'voter' | 'observer'`. В `ws.ts` используются type assertions (`as Participant['status']`, `as Participant['role']`), что обходит type-safety.
- **Влияние:** Type assertions маскируют потенциальные ошибки. Если бэкенд добавит новый статус или роль, TypeScript не поймает несоответствие.
- **Рекомендация:** Типизировать payload `participant_joined` с точными union-типами вместо `string`. Удалить type assertions в ws.ts.
- **Effort:** low

### [33] `RoomState` interface включает `timer` как обязательное поле, но тесты его часто опускают
- **Severity:** LOW
- **Файл:** web/src/state.ts:31-39, web/src/components/CardDeck/CardDeck.test.tsx:15-21
- **Проблема:** Interface `RoomState` определяет `timer: TimerState` как обязательное поле, но в тестах `CardDeck.test.tsx`, `observer.test.ts` и `ParticipantCard.test.tsx` создаются объекты `roomState.value` без поля `timer`. TypeScript strict mode не ловит это потому что тесты исключены из `tsconfig.json` (`"exclude": ["src/**/*.test.ts", "src/**/*.test.tsx"]`).
- **Влияние:** Тесты могут маскировать runtime-ошибки, связанные с отсутствием `timer`. Несогласованность между типами и тестовыми данными.
- **Рекомендация:** 1) Либо создать отдельный `tsconfig.test.json` включающий тесты для type-checking. 2) Либо сделать `timer` optional в интерфейсе (`timer?: TimerState`). 3) Либо обновить все тестовые хелперы чтобы включали `timer`.
- **Effort:** low

---

## [Производительность]

### [35] Каждое обновление roomState создает полную копию массива participants
- **Severity:** LOW
- **Файл:** web/src/ws.ts:124-270
- **Проблема:** Каждое событие (`vote_cast`, `vote_retracted`, `presence_changed`, `name_updated`, `role_updated`) создает новый объект `roomState` с помощью spread и `.map()` по всему массиву `participants`. При 50+ участниках это создаёт значительную нагрузку на GC, особенно при частых presence-событиях.
- **Влияние:** При большом количестве участников возможны микро-задержки из-за GC. На практике scrum poker редко имеет >20 участников, поэтому влияние минимально.
- **Рекомендация:** На текущем масштабе это приемлемо. При масштабировании рассмотреть immutable-библиотеку или точечные мутации с `signal.peek()` + `signal.value = { ... }`.
- **Effort:** high (при изменении), low (оставить как есть)

### [25] Footer делает fetch к /health при каждом рендере App
- **Severity:** LOW
- **Файл:** web/src/components/Footer/Footer.tsx:25-33
- **Проблема:** `Footer` вызывает `fetch('/health')` при каждом mount. При навигации между Home и Room (и обратно), Footer перемонтируется и повторяет запрос. На практике значение `build_time` не меняется в течение сессии.
- **Влияние:** Лишние HTTP-запросы при навигации. Минимальное влияние — запрос маленький и быстрый.
- **Рекомендация:** Кэшировать результат в модульной переменной или в сигнале на уровне state.ts, чтобы fetch выполнялся только один раз за сессию.
- **Effort:** low

---

## [Безопасность]

### [55] `navigator.clipboard.writeText` без проверки Permissions API
- **Severity:** MEDIUM
- **Файл:** web/src/components/Header/Header.tsx:14-18
- **Проблема:** `navigator.clipboard.writeText` вызывается без проверки поддержки API. В некоторых контекстах (HTTP без localhost, iframe без allow-clipboard) этот API недоступен и вызовет исключение. Текущий код обрабатывает reject Promise (`() => addToast('Failed to copy link', 'error')`), что покрывает async-ошибки, но `navigator.clipboard` может быть `undefined` — это вызовет TypeError до промиса.
- **Влияние:** Crash при клике "Copy Link" в non-secure context (HTTP). В production с HTTPS проблем нет.
- **Рекомендация:** Добавить проверку: `if (!navigator.clipboard) { addToast('Copy not available', 'error'); return; }`. Или использовать fallback через `document.execCommand('copy')`.
- **Effort:** low

### [30] Session ID хранится в localStorage без привязки к домену
- **Severity:** LOW
- **Файл:** web/src/state.ts:104-110
- **Проблема:** Session ID (`om-poker-session`) генерируется как 32 hex-символа и хранится в localStorage. Это достаточно для идентификации, но нет механизма ротации. Если пользователь делится браузером или использует shared компьютер, предыдущий session ID сохраняется навсегда.
- **Влияние:** Минимальный риск, т.к. session ID используется только для reconnect к комнатам, а комнаты эфемерны.
- **Рекомендация:** Принять текущее поведение. При необходимости добавить кнопку "Reset session" в будущем.
- **Effort:** low (нет действий)

---

## [Положительные аспекты — NOTE]

### [10] Отличная архитектура Modal компонента
- **Severity:** NOTE
- **Файл:** web/src/components/Modal/Modal.tsx
- **Проблема:** Не проблема. Modal использует нативный `<dialog>` элемент с `showModal()`, корректно управляет фокусом (сохранение и восстановление), поддерживает backdrop-click, dismiss/non-dismiss режимы, aria-атрибуты. Отличная реализация.
- **Влияние:** Положительно — надёжная основа для всех модальных окон.
- **Рекомендация:** Сохранить текущий подход.
- **Effort:** —

### [10] Хорошая система дизайн-токенов с защитой от регрессий
- **Severity:** NOTE
- **Файл:** web/src/tokens.css, web/src/design-tokens.test.ts
- **Проблема:** Не проблема. Автоматические тесты проверяют: наличие всех токенов, отсутствие hardcoded hex в компонентах, покрытие dark theme для каждого color/shadow токена, правильные конкретные значения для primary palette. Это превосходный подход.
- **Влияние:** Положительно — предотвращает регрессии дизайн-системы.
- **Рекомендация:** Сохранить и расширять по мере добавления новых токенов.
- **Effort:** —

### [10] Качественная реализация reconnect с exponential backoff и jitter
- **Severity:** NOTE
- **Файл:** web/src/ws.ts:62-66, 291-307
- **Проблема:** Не проблема. Reconnect логика использует exponential backoff (500ms base, 2x growth), 30% jitter, 10s cap, переход на slow polling (10s) после 30s timeout. Хорошо покрыто тестами. Соответствует best practices.
- **Влияние:** Положительно — надёжное восстановление соединения без thundering herd.
- **Рекомендация:** Сохранить текущий подход.
- **Effort:** —

### [10] Корректная реализация prefers-reduced-motion
- **Severity:** NOTE
- **Файл:** web/src/components/ParticipantCard/ParticipantCard.css:101-104
- **Проблема:** Не проблема. Flip-анимация карточек участников отключается через `@media (prefers-reduced-motion: reduce)`. Правильный подход к accessibility.
- **Влияние:** Положительно для пользователей с вестибулярными расстройствами.
- **Рекомендация:** Применить аналогичный подход к другим анимациям (toast-in, modal-in, connection-banner-slide-down).
- **Effort:** low

---

## Саммари
- Всего находок: 24
- CRITICAL (90-100): 0
- HIGH (70-89): 3
- MEDIUM (40-69): 13
- LOW (20-39): 4
- NOTE (0-19): 4
- Средний балл: 43.1
- Топ-3 проблемы:
  1. [75] Протокол WebSocket не документирован для timer и role событий
  2. [72] Очередь сообщений растёт без ограничений при отключении WebSocket
  3. [65] Нет защиты от XSS в отображаемых именах пользователей
