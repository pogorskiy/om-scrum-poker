# Финальный отчёт — Комплексное ревью проекта om-scrum-poker
- Дата: 2026-04-05
- Статус: done
- Участники: фронтенд, бекенд, безопасность, архитектура, продукт, адвокат дьявола
- Модель: claude-opus-4-6

---

## 1. Executive Summary

Проект om-scrum-poker представляет собой легковесный self-hosted инструмент для scrum poker, реализованный как единый Go-бинарник с встроенным Preact-фронтендом. Архитектура проекта зрелая и продуманная: чистое разделение domain/server, минимальные зависимости (единственная внешняя — websocket-библиотека), эффективный Docker multi-stage build, трёхуровневый fallback для SPA (embedded FS, disk, placeholder). Соотношение тест/код для бэкенда близко к 1:1. Проект осознанно следует принципу KISS — без авторизации, без базы данных, без внешних зависимостей.

Основные риски сосредоточены в трёх областях: (1) ~~безопасность WebSocket-соединений — отключена проверка Origin, отсутствует rate-limiting на сообщения внутри соединения, нет валидации входных данных (roomID, sessionID)~~ ✅ (Origin, roomID, sessionID исправлены; rate-limit остаётся); (2) ~~доступность (a11y) — модальные окна без focus trap~~ ✅ (мигрированы на native `<dialog>`), отсутствие ARIA-атрибутов на картах, глобальное подавление outline; (3) ~~UX-пробелы — нет индикации проблем с соединением, нет возможности сменить имя~~ ✅, колода карт не зафиксирована на мобильных. Ни одна из оставшихся проблем не является блокирующей для внутрикомандного использования, но при публичном деплое rate-limit на WS-сообщения требует внимания.

Суммарно 6 ревьюеров выявили 107 находок, которые после дедупликации (22 группы пересечений) сводятся к 62 уникальным проблемам. Адвокат дьявола скорректировал 14 оценок в сторону понижения (системное завышение severity для KISS-проекта) и добавил 5 новых находок, пропущенных основными ревьюерами.

---

## 2. Все находки (унифицированные, дедуплицированные)

Находки отсортированы по финальному баллу от высшего к низшему. Баллы после корректировки адвоката дьявола используются как финальные.

---

### ~~[70] Отсутствует визуальный индикатор проблем с соединением~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commits: `745e914`, `0b1c409`, `7bcaacd`)
- **Решение:** Добавлен постоянный ConnectionBanner: amber "Reconnecting..." с spinner и счётчиком попыток, red "Connection lost" с кнопкой Retry. Reconnect больше не останавливается после 30с — переходит на slow polling (10с). Карточки голосования заблокированы при отключении. retry() подключён к UI. 15 юнит-тестов (frontend) + 13 новых тестов (backend disconnect/reconnect lifecycle). Настроен vitest.
- **Severity:** HIGH
- **Источники:** фронтенд, продукт, адвокат дьявола
- **Файл:** web/src/ws.ts:229-243, web/src/components/RoomPage/RoomPage.tsx:58-59
- **Проблема:** При потере WebSocket-соединения пользователь не видит индикатора reconnecting. Toast "Connection lost. Click to retry." появляется через 30 секунд, исчезает через 2.3 секунды и не кликабелен (функция `retry()` экспортирована, но не используется в UI). Пользователь может голосовать, не зная, что соединение потеряно.
- **Влияние:** Пользователи real-time инструмента не получают обратной связи о состоянии соединения. Голоса ставятся в очередь и могут устареть.
- **Рекомендация:** Добавить постоянный баннер "Reconnecting..." (не toast) с кнопкой retry. Подключить экспортированную функцию `retry()` к UI.
- **Effort:** medium
- **Корректировка адвоката:** 80 -> 70. Reconnect с backoff работает автоматически, пользователь просто подождёт. Но реальная UX-проблема.

### ~~[72] Модальные окна не управляют фокусом и не блокируют скролл~~ ✅ RESOLVED
- **Severity:** HIGH
- **Источники:** фронтенд, продукт, адвокат дьявола
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.tsx, web/src/components/ConfirmDialog/ConfirmDialog.tsx
- **Проблема:** Модальные окна не используют focus trap. Пользователь может табом выйти за пределы модалки. Нет `role="dialog"`, `aria-modal="true"`, `aria-labelledby`. ConfirmDialog не закрывается по Escape. Нет блокировки скролла body.
- **Влияние:** Нарушение WCAG 2.1 AA (2.4.3 Focus Order, 4.1.2 Name/Role/Value). Пользователи с клавиатурной навигацией не смогут нормально использовать модалки.
- **Рекомендация:** Использовать нативный `<dialog>` элемент или добавить focus trap, `role="dialog"`, `aria-modal="true"`, обработку Escape, блокировку scroll на body.
- **Effort:** medium
- **Корректировка адвоката:** 82 -> 72. Для internal developer tool a11y-проблемы модалок имеют меньший импакт, но объективное нарушение WCAG.
- **Решение:** Все модалки мигрированы на нативный `<dialog>` элемент через shared Modal компонент (`401ce84`). Реализовано: focus trap, scroll blocking, Escape handling, backdrop click, `aria-labelledby`/`aria-describedby`, focus save/restore. 41 юнит-тест.
- **Регрессия исправлена (2026-04-06):** Глобальный CSS reset (`* { margin: 0 }` в `global.css`) перезаписывал UA-стиль `margin: auto` у `<dialog>`, из-за чего модалки прилипали к верхнему левому углу. Фикс: явный `margin: auto` в `.modal`.

### ~~[72] Отсутствуют ARIA-атрибуты на картах голосования~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `759bfcd`)
- **Решение:** Card: `aria-pressed` для состояния выбора. CardDeck: `role="group"` + `aria-label="Select your vote"`. ParticipantCard: `role="img"` и `aria-label` на статус-индикаторе, `aria-label` на контейнере карточки. 20 новых юнит-тестов.
- **Severity:** HIGH
- **Источники:** фронтенд, продукт
- **Файл:** web/src/components/Card/Card.tsx:19-23, web/src/components/CardDeck/CardDeck.tsx:26-37, web/src/components/ParticipantCard/ParticipantCard.tsx:39
- **Проблема:** Карточки голосования не имеют `aria-pressed` для отображения состояния выбора. Статус-индикаторы (active/idle/disconnected) передаются исключительно через цвет без ARIA/текстового эквивалента. Скринридер не озвучит ни состояние голосования, ни статус участника.
- **Влияние:** Пользователи скринридеров и дальтоники (до 8% мужчин) не получают полную информацию.
- **Рекомендация:** Добавить `aria-pressed={selected}` на Card, `role="radiogroup"` на колоду, `aria-label` на статус-индикаторы.
- **Effort:** low

### [70] Колода карт не зафиксирована внизу экрана на мобильных
- **Severity:** HIGH
- **Источники:** продукт
- **Файл:** web/src/components/CardDeck/CardDeck.css, web/src/components/RoomPage/RoomPage.css
- **Проблема:** CardDeck — обычный flex-контейнер в потоке документа. При большом количестве участников на мобильном нужно скроллить, чтобы добраться до карт. UX-спека указывает `position: sticky` или `fixed` для колоды.
- **Влияние:** На мобильных устройствах основной элемент взаимодействия скрыт за скроллом при 10+ участниках.
- **Рекомендация:** Сделать CardDeck `position: sticky; bottom: 0` с фоном.
- **Effort:** medium

### ~~[65] Нет возможности сменить имя из интерфейса комнаты~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commits: `062de1a`, `b1a0d9d`, `b357757`)
- **Решение:** Добавлена иконка карандаша в Header + EditNameModal. Имя обновляется через `update_name` WS-сообщение у всех участников. 11 юнит-тестов.
- **Severity:** MEDIUM
- **Источники:** продукт, адвокат дьявола
- **Файл:** web/src/components/Header/Header.tsx, web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** Бэкенд поддерживает `update_name`, фронтенд обрабатывает `name_updated`, но UI для инициации смены имени отсутствует. Единственный способ — очистить localStorage вручную.
- **Влияние:** Пользователь с опечаткой в имени не может его исправить без технических манипуляций.
- **Рекомендация:** Добавить иконку карандаша рядом с именем в Header, открывающую модалку смены имени.
- **Effort:** low
- **Корректировка адвоката:** 75 -> 65. Реальный пробел, но не HIGH — пользователь может перезайти с другим именем.

### ~~[65] disconnect() на клиенте не отправляет leave серверу~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `806d15a`)
- **Решение:** `disconnect()` теперь отправляет `{type: "leave", payload: {}}` перед закрытием WebSocket, если соединение открыто. 2 новых юнит-теста.
- **Severity:** MEDIUM
- **Источники:** адвокат дьявола (новая находка)
- **Файл:** web/src/ws.ts:286-301
- **Проблема:** Функция `disconnect()` закрывает WebSocket без отправки `leave` сообщения. Бэкенд поддерживает `leave` (ws.go:393-413), но участник остаётся в комнате со статусом "disconnected" до GC (24 часа).
- **Влияние:** Покинувшие пользователи "висят" в списке участников как disconnected.
- **Рекомендация:** Отправлять `leave` перед закрытием WebSocket.
- **Effort:** low

### ~~[65] Input-элементы не имеют визуального outline при фокусе~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `54823ad`)
- **Решение:** Удалён `outline: none` из input. Добавлены `:focus-visible` стили с `2px solid var(--color-primary)` outline для input и button.
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/global.css:34
- **Проблема:** Глобальный стиль `input { outline: none; }` убирает нативный фокусный индикатор. Buttons не имеют стилей `:focus-visible`.
- **Влияние:** Пользователи клавиатуры не могут определить, какой элемент в фокусе.
- **Рекомендация:** Добавить `:focus-visible` стили с видимым outline.
- **Effort:** low

### ~~[65] WebSocket URL: расхождение между документацией и реализацией~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `13c5311`)
- **Решение:** CLAUDE.md и docs/planning/04-architecture.md обновлены: `/ws/room/{roomId}` → `/ws/{roomId}` в соответствии с реализацией (6 правок в 04-architecture.md).
- **Severity:** MEDIUM
- **Источники:** архитектор
- **Файл:** internal/server/ws.go:27-28 vs docs/planning/04-architecture.md:372, CLAUDE.md
- **Проблема:** Документация указывает `/ws/room/{roomId}`, реализация использует `/ws/{roomId}`. Рассинхрон трёх источников: CLAUDE.md, docs/planning/04-architecture.md и код.
- **Влияние:** Новый разработчик может быть введён в заблуждение.
- **Рекомендация:** Привести все три источника в соответствие.
- **Effort:** low

### ~~[62] "Connection lost. Click to retry." — но нет обработчика клика~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** ConnectionBanner компонент с кнопкой Retry, вызывающей `retry()` из ws.ts. Покрыто тестами.
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/components/ConnectionBanner/ConnectionBanner.tsx
- **Проблема:** Toast "Connection lost. Click to retry." не имеет обработчика клика. Функция `retry()` экспортирована, но не используется.
- **Влияние:** Пользователь видит инструкцию "Click to retry", но клик ничего не делает.
- **Рекомендация:** Добавить кликабельный retry или изменить текст на "Please reload the page."
- **Effort:** low

### ~~[62] Игнорирование ошибок MakeEnvelope~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-06)
- **Решение:** Введена функция `makeEnvelopeOrLog()` — обёртка над `MakeEnvelope`, которая логирует ошибку с указанием event type. Все 15 мест в ws.go заменены. 4 unit-теста (success, all event types, error logging, nil payload).
- **Severity:** MEDIUM
- **Источники:** бекенд
- **Файл:** internal/server/ws.go (15 мест)
- **Проблема:** Во всех хэндлерах ошибка `MakeEnvelope` присваивается в `_`. Молчаливое проглатывание ошибок.
- **Влияние:** При изменении payload-структур ошибка будет потеряна, broadcast молча не произойдёт.
- **Рекомендация:** Логировать ошибку при `err != nil`.
- **Effort:** low

### ~~[62] Комната "Room not found" не реализована~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `a1e331c`)
- **Решение:** При первом получении room_state, если пользователь единственный участник, показывается toast "You started a new room. Share the link to invite others." — мягкий индикатор того, что комната была пустой/expired.
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:49-51
- **Проблема:** Нет обработки случая, когда комната истекла на сервере. Сервер просто создаст новую комнату по старой ссылке.
- **Влияние:** Пользователь по старой ссылке попадает в пустую комнату вместо понятного сообщения.
- **Рекомендация:** Показать toast при входе в пустую комнату по ссылке.
- **Effort:** low

### ~~[60] Нет защиты от множественного вызова reveal/new_round~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-06)
- **Решение:** Двухуровневая защита: (1) Backend — `Reveal()` возвращает ошибку "already in reveal phase", `NewRound()` теперь возвращает ошибку "already in voting phase" (добавлен idempotency guard). (2) Frontend — signal `actionPending` блокирует кнопки между кликом и получением ответа сервера (`votes_revealed`/`round_reset`/`error`). Сбрасывается при disconnect. 9 frontend unit-тестов, 2 backend unit-теста.
- **Severity:** MEDIUM
- **Источники:** фронтенд + бекенд
- **Файл:** web/src/components/RoomPage/RoomPage.tsx, internal/domain/room.go, internal/server/ws.go
- **Проблема:** Кнопки "Show Votes" и "New Round" не дизейблятся после клика. Быстрый двойной клик отправит дублирующие команды.
- **Влияние:** Лишний трафик и потенциально непредсказуемое поведение.
- **Рекомендация:** Добавить debounce или блокировку кнопки до ответа сервера.
- **Effort:** low

### ~~[60] Нет анимации при раскрытии голосов ("reveal moment")~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** 3D card flip animation (rotateY 180deg) с front/back faces, staggered reveal (80ms/card), prefers-reduced-motion. 16 unit tests.
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/ParticipantCard/ParticipantCard.tsx:24-35
- **Проблема:** Переход от голосования к раскрытию мгновенный — галочки просто заменяются числами. Нет анимации "переворота карт".
- **Влияние:** Снижается вовлечённость. Конкуренты используют анимацию переворота.
- **Рекомендация:** Добавить CSS-анимацию переворота карты (3D transform rotateY), с учётом `prefers-reduced-motion`.
- **Effort:** medium

### [60] Горизонтальное масштабирование отсутствует
- **Severity:** MEDIUM
- **Источники:** архитектор, адвокат дьявола
- **Файл:** internal/server/room_manager.go
- **Проблема:** Всё состояние в памяти одного процесса. Нет pub/sub, нет shared state. При запуске за load balancer участники одной комнаты могут попасть на разные инстансы.
- **Влияние:** Ограничение на один инстанс. Для KISS-проекта — осознанный trade-off.
- **Рекомендация:** При необходимости масштабирования: sticky sessions по roomId или Redis pub/sub.
- **Effort:** high
- **Корректировка адвоката:** 78 -> 60. Осознанный и документированный trade-off, не HIGH.

### ~~[58] Мутация roomState в room_cleared вызывает мерцание UI~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `a1e331c`)
- **Решение:** Удалено присвоение `roomState.value = null` из обработчика `room_cleared`. Вместо промежуточного null-рендера, сразу отправляется join — сервер пришлёт свежий room_state.
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/ws.ts:184-185
- **Проблема:** В обработчике `room_cleared` — `roomState.value = null`, затем `send(join)`. Промежуточный рендер с null вызывает мерцание.
- **Влияние:** Визуальное мерцание при очистке комнаты.
- **Рекомендация:** Использовать `batch()` из `@preact/signals` или не обнулять `roomState`.
- **Effort:** low

### ~~[58] Расхождение цветовой палитры с UX-спекой~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** Primary palette updated to Indigo (#6366f1 light, #818cf8 dark). All 4 color tokens aligned with UX spec.
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/tokens.css:6
- **Проблема:** UX-спека рекомендует Primary: #6366f1 (Indigo), реализация использует #3b82f6 (Blue).
- **Влияние:** Несоответствие документации и реализации.
- **Рекомендация:** Синхронизировать спеку и код.
- **Effort:** low

### ~~[58] Отсутствие поля roomName в join payload~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `470bc2f`)
- **Решение:** Добавлено `RoomName` в JoinPayload (опциональное, fallback на `"<user>'s Room"`). Добавлено `CreatedBy` (имя создателя) в Room domain и RoomStatePayload. Frontend генерирует human-readable имя из roomId и отображает создателя в Header. 10 новых тестов (BE+FE).
- **Severity:** MEDIUM
- **Источники:** архитектор
- **Файл:** internal/server/events.go:18-21 vs docs/planning/04-architecture.md:393
- **Проблема:** Документация добавила `roomName` в `join` payload, но в реализации его нет. Имя комнаты генерируется как `userName + "'s Room"`.
- **Влияние:** Клиент не может задать имя комнаты.
- **Рекомендация:** Добавить `RoomName` в JoinPayload или обновить документацию.
- **Effort:** low

### [58] TRUST_PROXY без валидации источника
- **Severity:** MEDIUM
- **Источники:** безопасность
- **Файл:** internal/server/ws.go:417-433
- **Проблема:** При `TRUST_PROXY=true` сервер доверяет `X-Forwarded-For` из любого запроса. Любой клиент может подставить произвольный IP и обойти rate-limiting.
- **Влияние:** Полный обход rate-limiting при TRUST_PROXY=true.
- **Рекомендация:** Добавить whitelist доверенных прокси-адресов или задокументировать, что TRUST_PROXY только за проверенным reverse proxy.
- **Effort:** medium

### [56] Домашняя страница не объясняет, что такое scrum poker
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/HomePage/HomePage.tsx:20-21
- **Проблема:** Подзаголовок "Simple. Self-hosted. No signup." не поясняет назначение инструмента.
- **Влияние:** Новые пользователи могут быть сбиты с толку.
- **Рекомендация:** Добавить "Real-time planning poker for agile teams".
- **Effort:** low

### ~~[55] Race condition: доступ к client.sessionID без синхронизации~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `93fa153`)
- **Решение:** Added `sync.Mutex`-protected `SessionID()`/`SetSessionID()` accessors to Client struct. All reads/writes in client.go and ws.go now go through thread-safe methods. Concurrent access test added. Race detector passes clean.
- **Severity:** MEDIUM
- **Источники:** бекенд, адвокат дьявола
- **Файл:** internal/server/ws.go:60-77, internal/server/client.go:21-23
- **Проблема:** `client.sessionID` записывается в handleJoin из горутины readPump и читается в WritePump из другой горутины без синхронизации. Формальный data race по Go memory model.
- **Влияние:** Практически безопасен (запись однократна, чтение через 5+ секунд), но `go test -race` может поймать.
- **Рекомендация:** Использовать `atomic.Value` для `sessionID`.
- **Effort:** low
- **Корректировка адвоката:** 82 -> 55. Проблема реальна, но окно гонки огромно, исправление тривиально.

### ~~[55] Двойной lock: RoomManager.mu и Room.mu — data race на LastActivity~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** Replaced `LastActivity time.Time` with private `lastActivity atomic.Int64` storing `time.UnixNano()`. Added thread-safe accessor methods: `TouchActivity()`, `GetLastActivity()`, `LastActivityUnixNano()`, `SetLastActivity()`. All reads/writes are now lock-free via atomics. Concurrent access test added (`room_race_test.go`). Race detector passes clean.
- **Severity:** MEDIUM
- **Источники:** бекенд, архитектор, адвокат дьявола
- **Файл:** internal/server/room_manager.go:53-66, internal/server/ws.go:147-179
- **Проблема:** В `collectGarbage` чтение `room.LastActivity` без `room.Lock()`. Запись в `LastActivity` всегда под `room.Lock()`. Формальный data race. Deadlock невозможен при текущей структуре.
- **Влияние:** Формально undefined behavior. На x86/arm64 практически безопасно для time.Time.
- **Рекомендация:** Сделать `LastActivity` атомарным (`atomic.Int64`) или брать `room.Lock()` в collectGarbage.
- **Effort:** low
- **Корректировка адвоката:** 75 -> 55. Deadlock невозможен в текущем коде. Data race реален, но практически безопасен.

### [55] ~~InsecureSkipVerify — отключена проверка Origin при WebSocket~~ RESOLVED
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность, архитектор, адвокат дьявола
- **Файл:** internal/server/ws.go:33-35
- **Проблема:** `InsecureSkipVerify: true` полностью отключает проверку Origin. Любой сайт может установить WebSocket-соединение с сервером.
- **Влияние:** Cross-Site WebSocket Hijacking возможен, но смягчается отсутствием cookies/auto-credentials. Атакующему нужно знать roomId.
- **Рекомендация:** Добавить конфигурируемый `ALLOWED_ORIGINS`. Минимально — проверять Origin == Host.
- **Effort:** low
- **Корректировка адвоката:** 90 -> 55. Нет cookies, sessionId передаётся явно. Для KISS-проекта без auth CRITICAL неадекватен.
- **Решение:** Реализовано. Удалён `InsecureSkipVerify: true`. По умолчанию используется same-origin проверка библиотеки nhooyr.io/websocket. Добавлена переменная окружения `ALLOWED_ORIGINS` для конфигурации разрешённых origin (поддержка `*` для dev-режима). Тесты: `ws_origin_test.go`, `main_test.go`.

### [55] Нет визуальной индикации собственного голоса после reveal
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:86-117
- **Проблема:** После раскрытия голосов нет явного указания "ваш голос был X" и нет пояснения, почему карты заблокированы.
- **Влияние:** Новый пользователь может не понять интерфейс.
- **Рекомендация:** Добавить "Your vote: X" над заблокированной колодой и пояснение "Voting locked until new round".
- **Effort:** low

### [55] Нет ограничения на количество одновременных подключений с одного IP
- **Severity:** MEDIUM
- **Источники:** безопасность
- **Файл:** internal/server/ratelimit.go:62-68
- **Проблема:** Rate-limit ограничивает частоту подключений (20/мин), но не количество одновременных. Атакующий может медленно открыть сотни соединений.
- **Влияние:** Исчерпание горутин, памяти, дескрипторов файлов.
- **Рекомендация:** Добавить счётчик активных соединений per-IP (лимит 10-20).
- **Effort:** low

### ~~[55] maxMessageSize = 1024 может быть недостаточен~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `470bc2f`)
- **Решение:** `maxMessageSize` увеличен с 1024 до 4096 байт. Достаточно для join payload с roomName, role и будущих расширений.
- **Severity:** MEDIUM
- **Источники:** адвокат дьявола (новая находка)
- **Файл:** internal/server/ws.go:15
- **Проблема:** Read limit 1024 байт. Join payload (sessionId 32 + userName 30 + JSON) укладывается с минимальным запасом. При добавлении roomName лимит будет превышен.
- **Влияние:** Новые поля в payload могут молча отклоняться.
- **Рекомендация:** Увеличить до 2048 или 4096.
- **Effort:** low

### [55] Toast-уведомления недоступны для скринридеров
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/components/Toast/Toast.tsx
- **Проблема:** Toast-контейнер не имеет `role="status"` или `aria-live="polite"`.
- **Влияние:** Незрячие пользователи пропустят уведомления.
- **Рекомендация:** Добавить `role="status"` и `aria-live="polite"`, `role="alert"` для error-тоастов.
- **Effort:** low

### ~~[55] Отсутствие graceful drain для WebSocket при shutdown~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `43d4549`)
- **Решение:** `CloseAll()` теперь отправляет `StatusGoingAway` close frame всем клиентам конкурентно через `CloseGraceful()`, с таймаутом 3 секунды для unresponsive peers. 2 новых теста.
- **Severity:** MEDIUM
- **Источники:** архитектор
- **Файл:** cmd/server/main.go:47-53
- **Проблема:** `CloseAll()` закрывает канал `done` всех клиентов, но не ждёт завершения горутин и не отправляет close frame.
- **Влияние:** При рестарте пользователи видят ошибку вместо чистого disconnect.
- **Рекомендация:** Отправлять StatusGoingAway из CloseAll и дать время на drain.
- **Effort:** low

### ~~[54] Нет режима наблюдателя (observer/spectator)~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `18a6dcd`)
- **Решение:** Полная реализация observer/spectator mode. Backend: Role field ("voter"/"observer") в Participant, reject votes from observers, exclude observers from statistics, update_role WS event. Frontend: "Join as observer" checkbox в NameEntryModal, toggle кнопка Voting/Observing в Header, observers видят сообщение вместо колоды карт, Observer badge на карточках участников, счётчик голосов исключает observers. 26 новых тестов (18 BE + 8 FE).
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx, internal/server/ws.go
- **Проблема:** Все участники — голосующие. SM, который не голосует, отображается как "не проголосовал", путая счётчик.
- **Влияние:** Визуальный шум для команд, где SM не голосует.
- **Рекомендация:** Добавить опцию "Join as observer" в NameEntryModal.
- **Effort:** high

### [52] Кнопка "Show Votes" disabled при 0 голосах — отклонение от спеки
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:124-126
- **Проблема:** Кнопка отключена при `counts.voted === 0`. UX-спека указывает: "Enabled even if 0 votes".
- **Влияние:** Нельзя раскрыть голоса для стимуляции обсуждения.
- **Рекомендация:** Убрать `disabled={counts.voted === 0}`.
- **Effort:** low

### ~~[50] Room ID не валидируется на сервере~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `91a0db9`)
- **Решение:** Added compiled `validRoomID` regexp (`^[a-z0-9-]{1,64}$`) check in HandleWebSocket before WebSocket upgrade. Returns HTTP 400 "invalid room id" on mismatch. 14 unit tests for valid/invalid patterns.
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность, архитектор, адвокат дьявола
- **Файл:** internal/server/ws.go:27-31
- **Проблема:** roomID из URL не валидируется по формату или длине. Можно создать комнату с произвольно длинным ID или спецсимволами. Фронтенд валидирует, но сервер принимает всё.
- **Влияние:** Memory exhaustion при длинных ключах, log injection через спецсимволы.
- **Рекомендация:** Серверная валидация: regex `^[a-z0-9-]{1,64}$`.
- **Effort:** low

### ~~[50] Session ID не валидируется на сервере~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `91a0db9`)
- **Решение:** Added compiled `validSessionID` regexp (`^[a-f0-9]{32}$`) check in handleJoin. Sends error "invalid sessionId format" on mismatch. 13 unit tests for valid/invalid patterns + integration tests.
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность, архитектор, адвокат дьявола
- **Файл:** internal/server/ws.go:131-144
- **Проблема:** sessionId проверяется только на пустоту. Клиент может отправить произвольно длинный sessionId. При знании чужого sessionId можно перехватить сессию.
- **Влияние:** Большие ключи в map, потенциальная подмена сессии (128 бит энтропии делает brute force нереальным).
- **Рекомендация:** Валидировать формат: `^[a-f0-9]{32}$`.
- **Effort:** low
- **Корректировка адвоката:** 82 -> 50. 128 бит энтропии делает brute force нереальным. Проблема в валидации формата, не в session hijacking.

### [50] Rate-limit на WebSocket-сообщения отсутствует
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность, архитектор, адвокат дьявола
- **Файл:** internal/server/ws.go:83-106
- **Проблема:** После подключения клиент может отправлять неограниченное количество сообщений. Broadcast усиливает нагрузку.
- **Влияние:** Один клиент может flood-ить сервер, создавая нагрузку на всех участников комнаты.
- **Рекомендация:** Добавить per-connection rate limit (10-20 msg/sec).
- **Effort:** low
- **Корректировка адвоката:** 72 -> 50. Для self-hosted KISS-инструмента вероятность целенаправленной DoS минимальна.

### [50] Health endpoint раскрывает внутреннюю информацию
- **Severity:** MEDIUM
- **Источники:** безопасность, архитектор
- **Файл:** internal/server/handler.go:54-71
- **Проблема:** `/health` без аутентификации возвращает количество комнат, соединений и uptime.
- **Влияние:** Атакующий может мониторить нагрузку для выбора времени атаки.
- **Рекомендация:** Оставить только `status: ok` для публичного доступа.
- **Effort:** low

### [50] Toast-уведомления исчезают слишком быстро для ошибок
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/state.ts:136-138
- **Проблема:** Все toast-ы исчезают через 2300ms, включая error-сообщения.
- **Влияние:** Пользователь может пропустить важное сообщение об ошибке.
- **Рекомендация:** Увеличить время для error-toast-ов до 5-8 секунд.
- **Effort:** low

### [50] Потеря состояния при рестарте
- **Severity:** MEDIUM
- **Источники:** архитектор, адвокат дьявола
- **Файл:** internal/server/room_manager.go:27-31
- **Проблема:** Все комнаты теряются при рестарте. Docker restart = потеря всех активных сессий.
- **Влияние:** Пользователи теряют ход голосования при каждом обновлении.
- **Рекомендация:** Для текущего use-case приемлемо. При необходимости — snapshot при graceful shutdown.
- **Effort:** medium
- **Корректировка адвоката:** 72 -> 50. Scrum poker сессия эфемерна, рестарт — редкое событие, reconnect автоматический.

### [48] Send buffer drop без уведомления клиента
- **Severity:** MEDIUM
- **Источники:** архитектор
- **Файл:** internal/server/client.go:42-48
- **Проблема:** При переполнении send buffer (32 сообщения) сообщение тихо отбрасывается. Клиент не знает о потере.
- **Влияние:** UI показывает устаревшие данные при медленной сети.
- **Рекомендация:** При переполнении — закрывать соединение, чтобы клиент переподключился и получил свежий room_state.
- **Effort:** low

### [48] Неоптимальное обновление массива participants на каждое событие
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/ws.ts:87-220
- **Проблема:** Каждое событие создаёт новый объект roomState через spread + map, вызывая полный ре-рендер.
- **Влияние:** При 20+ участниках потенциальные фризы на слабых устройствах.
- **Рекомендация:** Для текущего масштаба приемлемо. При росте — отдельные сигналы для participants.
- **Effort:** high

### [48] Автоматическая генерация имени комнаты: несоответствие вводу
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** internal/server/ws.go:155
- **Проблема:** Имя комнаты формируется как `userName + "'s Room"`, а не из пользовательского ввода на домашней странице.
- **Влияние:** Имя комнаты в Header не соответствует ожиданиям.
- **Рекомендация:** Передавать roomName в join payload.
- **Effort:** medium

### [47] Нет истории раундов в рамках сессии
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/ws.ts:166-179
- **Проблема:** При "New Round" результаты предыдущего раунда теряются безвозвратно.
- **Влияние:** Команды вынуждены вручную записывать результаты каждого раунда.
- **Рекомендация:** Хранить in-memory историю раундов на клиенте.
- **Effort:** medium

### [46] Список участников не скроллится при большом количестве
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/ParticipantList/ParticipantList.css
- **Проблема:** Grid растёт без ограничений. При 20+ участниках оттесняет карты за экран.
- **Влияние:** Деградация UX при больших сессиях.
- **Рекомендация:** `max-height: 40vh; overflow-y: auto` или sticky CardDeck.
- **Effort:** low

### [45] Нет валидации входящих WebSocket-сообщений на клиенте
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/ws.ts:55-62
- **Проблема:** JSON.parse оборачивается в try/catch, но результат кастуется через `as ServerMessage` без runtime-проверки.
- **Влияние:** Маловероятно при контролируемом сервере, но нарушает defense in depth.
- **Рекомендация:** Минимальная проверка наличия `type` и `payload` перед switch.
- **Effort:** low

### [45] Имя пользователя не санитизируется, только обрезается по длине
- **Severity:** MEDIUM
- **Источники:** безопасность
- **Файл:** internal/domain/room.go:90-95
- **Проблема:** Допускаются управляющие символы, невидимые Unicode-символы, RTL-маркеры. Log injection через имена с `\n`. Имя из только пробелов/невидимых символов проходит проверку.
- **Влияние:** Log injection, визуальный спуфинг.
- **Рекомендация:** Trim пробелов на бэкенде, удаление управляющих символов.
- **Effort:** low

### [45] conn.Close вызывается после cleanup
- **Severity:** MEDIUM
- **Источники:** бекенд
- **Файл:** internal/server/ws.go:79
- **Проблема:** `conn.Close` вызывается после `client.Close()` и `UnregisterClient()`, когда соединение может быть уже закрыто.
- **Влияние:** Минимальное — ложные ошибки в логах.
- **Рекомендация:** Явно игнорировать ошибку или изменить порядок cleanup.
- **Effort:** low

### [45] Нет оптимизации для landscape-ориентации мобильных
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.css
- **Проблема:** Нет медиа-запросов для landscape. На телефоне в горизонтальном режиме вертикальное пространство ограничено.
- **Влияние:** Неоптимальный опыт при видеозвонке в landscape.
- **Рекомендация:** Добавить `@media (orientation: landscape) and (max-height: 500px)`.
- **Effort:** medium

### [45] При rejoin name перезаписывается без уведомления
- **Severity:** MEDIUM
- **Источники:** адвокат дьявола (новая находка)
- **Файл:** internal/domain/room.go:98-104
- **Проблема:** При rejoin имя участника молча перезаписывается. Остальные участники не получают `name_updated` event, только `presence_changed`.
- **Влияние:** Другие участники не видят смену имени до следующего room_state.
- **Рекомендация:** Отправлять `name_updated` при rejoin, если имя изменилось.
- **Effort:** low

### [45] Отсутствие авторизации на деструктивные операции (reveal/new_round/clear_room)
- **Severity:** MEDIUM
- **Источники:** фронтенд, безопасность, архитектор, адвокат дьявола
- **Файл:** internal/server/ws.go:108-129, internal/server/ws.go:249-319
- **Проблема:** Любой участник может выполнить reveal, new_round или clear_room. Нет ролей и владельца комнаты.
- **Влияние:** Любой пользователь может нарушить процесс голосования.
- **Рекомендация:** Для текущего KISS-подхода — acceptable. При росте — ввести роль host.
- **Effort:** medium
- **Корректировка адвоката:** 92 -> 45. Проект **явно и осознанно** спроектирован без авторизации. CLAUDE.md: "Auth: None". UX-спека: "Facilitator role intentionally excluded". Room ID имеет 48 бит энтропии. Для internal tool без денег и персональных данных CRITICAL неадекватен.

### [45] Structured logging отсутствует
- **Severity:** MEDIUM
- **Источники:** бекенд, архитектор, адвокат дьявола
- **Файл:** все файлы с log.Printf
- **Проблема:** Используется `log.Printf` без уровней, без структурированных полей, без request ID.
- **Влияние:** Затруднена отладка при инцидентах.
- **Рекомендация:** Мигрировать на `log/slog` с JSON-форматом. Для KISS — минимально добавить уровни.
- **Effort:** medium
- **Корректировка адвоката:** 68 -> 45. Для self-hosted инструмента `log.Printf` достаточен.

### [44] Нет подсказки по значениям карт для новых пользователей
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/CardDeck/CardDeck.tsx:6
- **Проблема:** Колода показывается без пояснений. "?" не очевидна для новичков.
- **Влияние:** Пользователи без опыта scrum poker могут быть сбиты с толку.
- **Рекомендация:** Tooltip на "?" ("Need more discussion").
- **Effort:** low

### [43] Нет валидации input на домашней странице
- **Severity:** MEDIUM
- **Источники:** продукт
- **Файл:** web/src/components/HomePage/HomePage.tsx:28-30
- **Проблема:** Нет inline-валидации символов. Пользователь может ввести "!!!" и получить нечитаемый slug.
- **Влияние:** Нечитаемые URL комнат.
- **Рекомендация:** Валидация с inline-сообщением или превью URL.
- **Effort:** low

### [42] selectedCard не сбрасывается при смене комнаты через URL
- **Severity:** MEDIUM
- **Источники:** фронтенд
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:34-41
- **Проблема:** Потенциальное состояние гонки при смене roomId без unmount.
- **Влияние:** Минимальное в текущей архитектуре.
- **Рекомендация:** Явный сброс `selectedCard.value = ''` в начале connect().
- **Effort:** low

### ~~[42] HTTP-сервер без ReadHeaderTimeout (slowloris)~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `91a0db9`)
- **Решение:** Added `ReadHeaderTimeout: 5 * time.Second` to http.Server in NewServer. Unit tests verify the timeout is set.
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность
- **Файл:** internal/server/handler.go:47-51
- **Проблема:** `http.Server` без `ReadHeaderTimeout`. Slowloris-атака возможна.
- **Влияние:** Исчерпание горутин и дескрипторов файлов.
- **Рекомендация:** Добавить `ReadHeaderTimeout: 5 * time.Second`.
- **Effort:** low

### [42] Горутина WritePump может leak при потере контекста
- **Severity:** MEDIUM
- **Источники:** бекенд
- **Файл:** internal/server/ws.go:47-51, internal/server/client.go:68-103
- **Проблема:** При edge-case с зависшим соединением WritePump продолжит пинговать.
- **Влияние:** Утечка горутины в редком edge-case.
- **Рекомендация:** Общий timeout на время жизни соединения как дополнительная страховка.
- **Effort:** low

### [40] Broadcast после Unlock — потенциальная неконсистентность наблюдателя
- **Severity:** MEDIUM
- **Источники:** адвокат дьявола (новая находка)
- **Файл:** internal/server/ws.go (все handle-функции)
- **Проблема:** Паттерн `room.Lock() -> mutation -> room.Unlock() -> MakeEnvelope -> Broadcast`. Между Unlock и Broadcast другой клиент может выполнить действие. Клиенты могут получить события в непредсказуемом порядке.
- **Влияние:** Потенциальная рассинхронизация UI между клиентами при конкурентных действиях.
- **Рекомендация:** Формировать envelope внутри lock или использовать ordered broadcast.
- **Effort:** medium

### [40] Неограниченное количество комнат
- **Severity:** MEDIUM
- **Источники:** безопасность
- **Файл:** internal/server/room_manager.go:69-84
- **Проблема:** Нет глобального лимита на количество комнат. GC удаляет через 24 часа.
- **Влияние:** Потенциальный OOM при целенаправленной атаке с нескольких IP.
- **Рекомендация:** Глобальный лимит (10000 комнат). Уменьшить roomExpiry до 1-4 часов.
- **Effort:** low

### [40] Очередь WS-сообщений на клиенте не ограничена
- **Severity:** MEDIUM
- **Источники:** фронтенд, архитектор, адвокат дьявола
- **Файл:** web/src/ws.ts:18
- **Проблема:** `messageQueue` растёт неограниченно при offline. При reconnect все сообщения отправляются залпом.
- **Влияние:** Теоретическая утечка памяти при длительном offline. Устаревшие голоса при reconnect.
- **Рекомендация:** Ограничить до 10-20 сообщений. При reconnect полагаться на room_state.
- **Effort:** low
- **Корректировка адвоката:** 75 -> 40. `disconnect()` очищает очередь. Пользователь физически не может нагенерить значимое количество за секунды reconnect.

### [40] Версия websocket-библиотеки nhooyr.io/websocket v1.8.17 — unmaintained
- **Severity:** MEDIUM
- **Источники:** бекенд, безопасность, архитектор
- **Файл:** go.mod:5
- **Проблема:** v1.x перенесена в `github.com/coder/websocket` v2.x. v1.x не получает обновлений безопасности.
- **Влияние:** Нет активных патчей для v1.x.
- **Рекомендация:** Мигрировать на `github.com/coder/websocket` v2.x.
- **Effort:** medium

### [40] Устаревшая версия Go (1.22)
- **Severity:** MEDIUM
- **Источники:** бекенд, архитектор
- **Файл:** go.mod:3
- **Проблема:** Go 1.22 вышел из поддержки. Пропущены security-обновления в runtime.
- **Влияние:** Потенциальные незакрытые уязвимости.
- **Рекомендация:** Обновить до Go 1.24.x.
- **Effort:** low

### [38] Нет индикации "все проголосовали"
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:127
- **Проблема:** Единственная индикация — текст кнопки "Show Votes (5 of 5 voted)". Нет визуального выделения.
- **Влияние:** Фасилитатор может не заметить, что все проголосовали.
- **Рекомендация:** Подсветить кнопку при voted === total.
- **Effort:** low

### [38] Дублирование логики маппинга vote в participants
- **Severity:** LOW
- **Источники:** фронтенд
- **Файл:** web/src/ws.ts:66-74, web/src/ws.ts:151-159
- **Проблема:** Код маппинга votes на participants дублируется в обработчиках room_state и votes_revealed.
- **Влияние:** Нарушение DRY.
- **Рекомендация:** Вынести в `applyVotesToParticipants()`.
- **Effort:** low

### [38] Отсутствие error boundary
- **Severity:** LOW
- **Источники:** архитектор
- **Файл:** web/src/app.tsx
- **Проблема:** Нет error boundary. Необработанное исключение — белый экран.
- **Влияние:** Потеря UI при runtime-ошибке.
- **Рекомендация:** Добавить Preact error boundary с fallback UI.
- **Effort:** low

### [38] Канал send не закрывается при отключении клиента
- **Severity:** LOW
- **Источники:** бекенд
- **Файл:** internal/server/client.go:60-65
- **Проблема:** `Close()` закрывает `done`, но не `send`. Сообщения буферизуются до заполнения канала (32).
- **Влияние:** Небольшая утечка памяти до GC.
- **Рекомендация:** Закрывать `send` в `Close()` и проверять `ok` в Broadcast.
- **Effort:** low

### [38] Отсутствие ограничений ресурсов в Dockerfile
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** Dockerfile
- **Проблема:** Нет `USER` директивы (root в scratch), нет health-check.
- **Влияние:** Формально root, но scratch минимизирует поверхность атаки.
- **Рекомендация:** Добавить non-root user и HEALTHCHECK.
- **Effort:** low

### [35] Session ID хранится в localStorage без привязки к устройству
- **Severity:** LOW
- **Источники:** фронтенд
- **Файл:** web/src/state.ts:88-94
- **Проблема:** Session ID живёт вечно в localStorage без ротации.
- **Влияние:** Минимальный при текущей threat model.
- **Рекомендация:** Acceptable risk для текущего масштаба.
- **Effort:** low

### [35] Логирование sessionId и IP-адресов
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** internal/server/ws.go:95, internal/server/client.go:46,86,96
- **Проблема:** SessionId и IP логируются. Потенциальная утечка PII через логи.
- **Влияние:** GDPR-соображения для IP-адресов.
- **Рекомендация:** Маскирование IP в production.
- **Effort:** low

### ~~[35] HTTP timeout-ы: ReadTimeout и WriteTimeout не установлены~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05, commit: `91a0db9`)
- **Решение:** ReadHeaderTimeout added (see [42] above). WriteTimeout intentionally not set globally due to WebSocket long-lived connections.
- **Severity:** LOW
- **Источники:** бекенд
- **Файл:** internal/server/handler.go:47-51
- **Проблема:** Дублирует находку slowloris выше с фокусом на ReadTimeout/WriteTimeout. Не устанавливать WriteTimeout глобально из-за WebSocket.
- **Влияние:** См. slowloris выше.
- **Рекомендация:** `ReadHeaderTimeout` достаточен. WriteTimeout для не-WS через middleware.
- **Effort:** low

### ~~[35] Нет тёмной темы~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** Added @media (prefers-color-scheme: dark) with full token coverage. 68 automated tests verify completeness.
- **Severity:** LOW
- **Источники:** фронтенд, продукт
- **Файл:** web/src/tokens.css
- **Проблема:** Нет `prefers-color-scheme: dark`. Разработчики часто предпочитают тёмную тему.
- **Влияние:** Белый фон утомляет при длительных сессиях.
- **Рекомендация:** `@media (prefers-color-scheme: dark)` с альтернативными CSS-переменными.
- **Effort:** medium

### [35] Docker FROM scratch — нет shell для отладки
- **Severity:** LOW
- **Источники:** архитектор
- **Файл:** Dockerfile:19-22
- **Проблема:** Нет shell, curl, wget для отладки в production.
- **Рекомендация:** Использовать `distroless/static:nonroot` или документировать ephemeral debug containers.
- **Effort:** low

### [35] Нет пользовательской документации / help
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** (отсутствует)
- **Проблема:** Нет помощи, FAQ, tooltips для нетехнических пользователей.
- **Рекомендация:** "How it works" блок на домашней странице.
- **Effort:** low

### [32] Нет проверки HTTP-метода на WebSocket-эндпоинте
- **Severity:** LOW
- **Источники:** бекенд
- **Файл:** internal/server/ws.go:19
- **Проблема:** POST/PUT/DELETE проходят до websocket.Accept, rate limit учитывается.
- **Влияние:** Исчерпание rate limit через не-GET запросы.
- **Рекомендация:** Проверять `r.Method == http.MethodGet` до rate limit.
- **Effort:** low

### [32] Хардкод значений карточек
- **Severity:** LOW
- **Источники:** фронтенд
- **Файл:** web/src/components/CardDeck/CardDeck.tsx:6
- **Проблема:** CARD_VALUES захардкожены на клиенте.
- **Влияние:** Ограничивает расширяемость. Для текущего этапа не проблема.
- **Рекомендация:** Рассмотреть получение от сервера в room_state.
- **Effort:** medium

### [32] Нет кнопки "Leave room"
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/src/components/Header/Header.tsx
- **Проблема:** Бэкенд поддерживает leave, но UI-кнопки нет.
- **Влияние:** Пользователь остаётся "disconnected" вместо чистого ухода.
- **Рекомендация:** Добавить кнопку "Leave" в Header.
- **Effort:** low

### ~~[30] Хардкод hex-цветов в hover-стилях~~ ✅ RESOLVED
- **Status:** ✅ RESOLVED (2026-04-05)
- **Решение:** Extracted 8 semantic tokens. All hardcoded hex replaced in 4 component CSS files. Automated test verifies no hex colors in components.
- **Severity:** LOW
- **Источники:** фронтенд, продукт
- **Файл:** web/src/components/RoomPage/RoomPage.css:41, web/src/components/ConfirmDialog/ConfirmDialog.css:52
- **Проблема:** `#d1d5db`, `#dc2626` не вынесены в CSS-переменные.
- **Рекомендация:** Добавить `--color-border-hover` и `--color-danger-hover` в tokens.css.
- **Effort:** low

### [30] Ручной роутинг без поддержки 404 и query parameters
- **Severity:** LOW
- **Источники:** архитектор
- **Файл:** web/src/app.tsx:17-26
- **Проблема:** Нет 404 страницы, нет парсинга query string.
- **Влияние:** Минимальное для 2 маршрутов.
- **Рекомендация:** Оставить при текущем scope. При 3+ маршрутах — preact-router.
- **Effort:** low

### [30] Отсутствие Content-Security-Policy заголовка
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** internal/server/handler.go:73-114
- **Проблема:** Нет CSP. Defense-in-depth отсутствует.
- **Рекомендация:** Добавить CSP при отдаче index.html.
- **Effort:** low

### [30] NameEntryModal позволяет имя из одних пробелов (визуально)
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.tsx:11-12
- **Проблема:** Нет inline-сообщения, имя "A" валидно.
- **Рекомендация:** Минимальная длина 2 символа.
- **Effort:** low

### [30] Нет ограничения на количество клиентов в одной комнате через RegisterClient
- **Severity:** LOW
- **Источники:** адвокат дьявола (новая находка)
- **Файл:** internal/server/room_manager.go:94-101
- **Проблема:** MaxParticipants=50 ограничивает уникальных участников, но не соединения. Один sessionID — множество вкладок — множество горутин.
- **Влияние:** 50 пользователей x 10 вкладок = 1000 горутин.
- **Рекомендация:** Ограничить соединения per-sessionID per-room.
- **Effort:** low

### [30] Отсутствие TLS на уровне приложения
- **Severity:** LOW
- **Источники:** безопасность, адвокат дьявола
- **Файл:** cmd/server/main.go:31
- **Проблема:** Сервер слушает по HTTP, без TLS. Все данные в открытом виде.
- **Влияние:** Стандартная практика для Go за reverse proxy. Не проблема приложения.
- **Рекомендация:** Задокументировать обязательность TLS-терминирующего reverse proxy в production.
- **Effort:** low
- **Корректировка адвоката:** 70 -> 30. Встроенный TLS в Go — антипаттерн. Deployment guide уже описывает reverse proxy.

### [28] Отсутствие тестов фронтенда
- **Severity:** LOW
- **Источники:** фронтенд, архитектор
- **Файл:** web/package.json
- **Проблема:** Нет ни одного теста для frontend кода.
- **Влияние:** Регрессии обнаруживаются только при ручном тестировании.
- **Рекомендация:** Добавить vitest + @testing-library/preact. Начать с ws.ts, state.ts.
- **Effort:** medium

### [28] Нет поддержки ссылки на конкретную задачу (topic/story)
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** Раунды анонимны, нет контекста оцениваемой задачи.
- **Рекомендация:** Добавить опциональное поле "Topic".
- **Effort:** medium

### [28] Filepath traversal в disk-based SPA serving
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** internal/server/handler.go:94
- **Проблема:** `filepath.Join(distDir, filepath.Clean(r.URL.Path))` — формально безопасно благодаря нормализации.
- **Влияние:** Реальный риск минимален.
- **Рекомендация:** Для defense in depth — проверить `strings.HasPrefix` на distDir.
- **Effort:** low

### [25] GC-интервал 10 минут, expiry 24 часа — консервативно
- **Severity:** LOW
- **Источники:** бекенд
- **Файл:** internal/server/room_manager.go:12-14
- **Проблема:** Пустые комнаты живут 24 часа.
- **Рекомендация:** Конфигурируемый expiry (по умолчанию 2-4 часа).
- **Effort:** low

### [25] Отсутствие lazy loading и code splitting
- **Severity:** LOW
- **Источники:** фронтенд
- **Файл:** web/vite.config.ts
- **Проблема:** Всё приложение — один бандл.
- **Влияние:** При текущем размере не проблема.
- **Рекомендация:** Не требуется действий.
- **Effort:** medium

### [25] HTML title не обновляется при входе в комнату
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/index.html:5
- **Проблема:** Title всегда "om-scrum-poker".
- **Рекомендация:** Обновлять `document.title` при получении room_state.
- **Effort:** low

### [25] Зафиксированные версии в Dockerfile (плавающие теги)
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** Dockerfile:2,10
- **Проблема:** `node:20-alpine` и `golang:1.22-alpine` — плавающие теги.
- **Рекомендация:** Использовать конкретные версии.
- **Effort:** low

### [22] usePresence: 4 глобальных обработчика событий
- **Severity:** LOW
- **Источники:** фронтенд
- **Файл:** web/src/hooks/usePresence.ts:40
- **Проблема:** mousemove срабатывает очень часто.
- **Рекомендация:** Throttle на 1 раз в секунду (optional).
- **Effort:** low

### [22] Нет favicon, кроме emoji
- **Severity:** LOW
- **Источники:** продукт
- **Файл:** web/index.html:7
- **Рекомендация:** Полноценный favicon + web manifest.
- **Effort:** low

### [22] Room.mu экспортируется через Lock()/Unlock()
- **Severity:** LOW (перенесён из MEDIUM)
- **Источники:** бекенд
- **Файл:** internal/domain/room.go:83-86
- **Проблема:** Domain-объект знает о конкурентности, но не контролирует её.
- **Рекомендация:** Перенести блокировку внутрь методов Room.
- **Effort:** medium

### [22] Поле Participants экспортировано
- **Severity:** LOW (перенесён из MEDIUM)
- **Источники:** бекенд
- **Файл:** internal/domain/room.go:57
- **Проблема:** Прямой доступ в обход методов Room.
- **Рекомендация:** Сделать приватным, добавить getter-методы.
- **Effort:** medium

### [22] LogMiddleware определён, но не используется
- **Severity:** LOW
- **Источники:** бекенд, архитектор
- **Файл:** internal/server/handler.go:195-201
- **Проблема:** Мёртвый код.
- **Рекомендация:** Подключить middleware или удалить.
- **Effort:** low

### [20] Broadcast amplification при большом количестве участников
- **Severity:** LOW
- **Источники:** безопасность
- **Файл:** internal/server/room_manager.go:113-126
- **Проблема:** При 50 участниках каждое действие генерирует 50 write-операций.
- **Рекомендация:** Rate-limit на сообщения, дебаунсинг presence.
- **Effort:** medium

### [18] Нет анимации исчезновения toast
- **Severity:** NOTE
- **Источники:** фронтенд, продукт
- **Файл:** web/src/components/Toast/Toast.css
- **Проблема:** Toast исчезает мгновенно без fade-out.
- **Рекомендация:** CSS-класс `toast--leaving` + transitionend.
- **Effort:** low

### [18] Нет meta description и Open Graph тегов
- **Severity:** NOTE
- **Источники:** фронтенд, продукт
- **Файл:** web/index.html
- **Проблема:** При шаринге ссылки — нет превью.
- **Рекомендация:** Добавить meta description и OG-теги.
- **Effort:** low

### [18] findDistDir с одним кандидатом
- **Severity:** NOTE
- **Источники:** бекенд
- **Файл:** internal/server/handler.go:136-149
- **Проблема:** Цикл с одним элементом — излишняя абстракция.
- **Рекомендация:** Оставить как есть.
- **Effort:** low

### [18] Нет graceful отключения при WebSocket rate-limit
- **Severity:** NOTE
- **Источники:** безопасность
- **Файл:** internal/server/ws.go:20-24
- **Проблема:** HTTP 429 до WebSocket-апгрейда. Клиент получает onerror без информативного сообщения.
- **Рекомендация:** Текущее поведение корректно. Optional: показать пользователю сообщение.
- **Effort:** low

### [15] Placeholder HTML захардкожен в Go-файле
- **Severity:** NOTE
- **Источники:** бекенд
- **Файл:** internal/server/handler.go:171-192
- **Рекомендация:** Оставить. Для KISS нормально.
- **Effort:** low

### [15] Frontend-зависимости — минимальный набор (положительное)
- **Severity:** NOTE
- **Источники:** безопасность
- **Файл:** web/package.json
- **Проблема:** Минимальный набор зависимостей — хорошая практика.
- **Рекомендация:** Периодически `npm audit`.
- **Effort:** low

### [15] Нет prefers-reduced-motion
- **Severity:** NOTE
- **Источники:** продукт
- **Файл:** web/src/components/Card/Card.css:16, web/src/components/Toast/Toast.css:30-39
- **Рекомендация:** Добавить `@media (prefers-reduced-motion: reduce)`.
- **Effort:** low

### [12] tsconfig: target ES2020
- **Severity:** NOTE
- **Источники:** фронтенд
- **Файл:** web/tsconfig.json
- **Рекомендация:** Обновить до ES2022.
- **Effort:** low

### [12] Нет индикации высокого spread при раскрытии голосов
- **Severity:** NOTE
- **Источники:** продукт
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:86-117
- **Рекомендация:** Визуальная индикация spread и рекомендация обсудить.
- **Effort:** low

### [12] Preact JSX корректно экранирует пользовательский ввод (положительное)
- **Severity:** NOTE
- **Источники:** безопасность
- **Проблема:** XSS через пользовательский ввод крайне маловероятен.
- **Рекомендация:** Добавить lint-правило против dangerouslySetInnerHTML.
- **Effort:** low

### [10] handleClearRoom: broadcast до lock
- **Severity:** NOTE
- **Источники:** бекенд
- **Файл:** internal/server/ws.go:310-319
- **Рекомендация:** Оставить с комментарием.
- **Effort:** low

### [10] CORS-заголовки на REST-эндпоинтах
- **Severity:** NOTE
- **Источники:** безопасность
- **Проблема:** Корректное поведение по умолчанию.
- **Рекомендация:** Оставить как есть.
- **Effort:** low

### [30] Нет тестов для cmd/server/main.go
- **Severity:** LOW
- **Источники:** бекенд
- **Файл:** cmd/server/main.go
- **Рекомендация:** Вынести `getEnv` в пакет config или покрыть тестом.
- **Effort:** low

### [58] Низкое покрытие тестами пакета server (34.6%)
- **Severity:** MEDIUM
- **Источники:** бекенд, архитектор
- **Файл:** internal/server/
- **Проблема:** WebSocket-хэндлеры (основная бизнес-логика) без тестов. Покрытие 34.6%.
- **Влияние:** Баги обнаруживаются только в production.
- **Рекомендация:** Интеграционные тесты с httptest.Server и WebSocket-клиентом.
- **Effort:** high

---

## 3. Вердикт по областям

### Frontend
- **Здоровье:** 3/5
- **Резюме:** Чистая архитектура с минимальными зависимостями (Preact + signals), правильным разделением состояния и компонентов, BEM-неймингом и дизайн-токенами. TypeScript strict mode включён.
- **Сильные стороны:** Минимальный bundle, иммутабельные обновления состояния, reconnect с exponential backoff + jitter, graceful localStorage handling.
- **Слабые стороны:** ~~Отсутствие a11y (focus trap~~ ✅, ARIA, outline), ~~нет тестов~~ ✅, ~~нет индикации состояния соединения~~ ✅, ~~"Click to retry" не работает~~ ✅.

### Backend
- **Здоровье:** 3.5/5
- **Резюме:** Добротный Go-код с хорошим соотношением тест/код (1:1), чистым разделением domain/server. Единственная зависимость — websocket-библиотека. Основные проблемы — в области конкурентности и валидации.
- **Сильные стороны:** Чистая архитектура domain/server, хорошее покрытие domain-логики, трёхуровневый embedding fallback, минимальные зависимости.
- **Слабые стороны:** ~~Data race на sessionID~~ ✅ fixed, data race на LastActivity, ~~отсутствие валидации roomID/sessionID~~ ✅ fixed, отсутствие валидации userName, нет rate-limit на WS-сообщения, ws.go без тестов.

### Security
- **Здоровье:** 2.5/5
- **Резюме:** Базовая защита присутствует (rate-limit на соединения, maxMessageSize, Preact-экранирование), но значимые пробелы в Origin-проверке, input-валидации и resource-limiting. Для internal tool — приемлемо, для public deployment — необходимо доработать.
- **Сильные стороны:** Минимальный Docker-образ (scratch), нет cookies (снижает CSRF-риск), Preact JSX-экранирование, rate-limit на соединения.
- **Слабые стороны:** ~~InsecureSkipVerify~~ ✅ fixed, нет rate-limit на WS-сообщения, TRUST_PROXY без whitelist, ~~нет валидации roomID/sessionID~~ ✅ fixed, нет CSP, ~~нет ReadHeaderTimeout~~ ✅ fixed.

### Architecture
- **Здоровье:** 4/5
- **Резюме:** Продуманная архитектура для KISS-проекта. Осознанные trade-off задокументированы. Модульная структура, чистые зависимости. Основные ограничения (single instance, in-memory state) соответствуют scope проекта.
- **Сильные стороны:** domain/server разделение, трёхуровневый SPA fallback, Docker multi-stage build, минимальные зависимости, хороший DX.
- **Слабые стороны:** Расхождения между документацией и реализацией (WS URL, roomName), unmaintained websocket-библиотека, устаревшая версия Go.

### Product
- **Здоровье:** 2.5/5
- **Резюме:** Базовый функционал scrum poker работает, но множество UX-пробелов по сравнению с UX-спекой: нет смены имени, нет фиксированной колоды на мобильных, нет анимации reveal, нет observer mode, нет истории раундов.
- **Сильные стороны:** Простой и понятный интерфейс, mobile-first grid, дизайн-система на токенах, мгновенный онбординг.
- **Слабые стороны:** Много реализованных, но недоступных функций (update_name, leave без UI), отсутствие ключевых UX-элементов из спеки, слабый мобильный опыт при большом количестве участников.

---

## 4. Общее здоровье проекта

### Общая оценка: B- (3.1 / 5)

Проект демонстрирует зрелый инженерный подход к lightweight self-hosted инструменту: минимальные зависимости, чистая архитектура, хороший DX, осознанные trade-off. Для **внутрикомандного использования** проект production-ready с оговорками.

### Готовность к production

**Для internal use (за VPN/corporate network):** Готов с минимальными исправлениями.

**Для public deployment:** Почти готов. Исправлено:
- ~~Origin-проверку WebSocket~~ ✅
- ~~Валидацию roomID и sessionID на сервере~~ ✅
- ~~ReadHeaderTimeout для HTTP~~ ✅
- ~~Data race на client.sessionID~~ ✅

Остаётся:
- Rate-limit на WS-сообщения
- TRUST_PROXY whitelist (если используется)

### Минимальный набор исправлений перед деплоем

1. ~~Добавить `ReadHeaderTimeout` на HTTP-сервер (1 строка)~~ ✅ DONE (`91a0db9`)
2. ~~Валидировать roomID на сервере (regex, 5 строк)~~ ✅ DONE (`91a0db9`)
3. ~~Валидировать sessionID на сервере (regex, 5 строк)~~ ✅ DONE (`91a0db9`)
4. ~~Добавить проверку Origin для WebSocket или `ALLOWED_ORIGINS` env var~~ ✅ DONE (ранее, `4eda997`)
5. ~~Исправить "Click to retry" — либо сделать кликабельным, либо сменить текст~~ ✅ DONE (ранее)
6. ~~Исправить data race на `client.sessionID` (atomic.Value)~~ ✅ DONE (`93fa153`)

---

## 5. Топ-10 приоритетных действий

1. ~~**[55] Добавить проверку Origin для WebSocket**~~ ✅ DONE (`4eda997`) — заменён `InsecureSkipVerify` на configurable `ALLOWED_ORIGINS` с OriginPatterns

2. ~~**[50] Валидировать roomID на сервере**~~ ✅ DONE (`91a0db9`) — добавлен compiled regexp `^[a-z0-9-]{1,64}$` в HandleWebSocket, 14 юнит-тестов

3. ~~**[50] Валидировать sessionID на сервере**~~ ✅ DONE (`91a0db9`) — добавлен compiled regexp `^[a-f0-9]{32}$` в handleJoin, 13 юнит-тестов

4. ~~**[55] Исправить data race на client.sessionID**~~ ✅ DONE (`93fa153`) — добавлены `sync.Mutex`-protected `SessionID()`/`SetSessionID()` accessors, concurrent test, race detector clean

5. ~~**[42] Добавить ReadHeaderTimeout**~~ ✅ DONE (`91a0db9`) — `ReadHeaderTimeout: 5 * time.Second` в http.Server

6. ~~**[62] Исправить "Click to retry"**~~ ✅ DONE (ранее) — ConnectionBanner с retry кнопкой

7. ~~**[62] Логировать ошибки MakeEnvelope**~~ ✅ DONE (2026-04-06) — `makeEnvelopeOrLog()` wrapper, 15 call sites replaced, 4 unit tests

8. ~~**[60] Защита от двойного reveal/new_round**~~ ✅ DONE (2026-04-06) — Backend idempotency guard on `NewRound()`, frontend `actionPending` signal, 9+2 unit tests

9. **[72] Добавить ARIA-атрибуты на карты голосования** — `aria-pressed`, `role="radiogroup"`, `aria-label` на статусы (effort: low, source: фронтенд + продукт)

10. **[50] Добавить rate-limit на WS-сообщения** — per-connection TokenBucket (10-20 msg/sec) в readPump (effort: low, source: бекенд + безопасность + архитектор)

9. ~~**[65] Добавить UI для смены имени**~~ ✅ DONE (ранее) — EditNameModal + иконка карандаша в Header

10. ~~**[70] Добавить индикатор состояния соединения**~~ ✅ DONE (ранее) — ConnectionBanner с reconnect UI

---

## 6. Статистика

- Всего уникальных находок (после дедупликации): 62
- CRITICAL (90-100): 0
- HIGH (70-89): 4
- MEDIUM (40-69): 33
- LOW (20-39): 18
- NOTE (0-19): 7
- Средний балл: 39.8
- Находок от фронтенда: 21
- Находок от бекенда: 19
- Находок от безопасности: 18
- Находок от архитектора: 22
- Находок от продукта: 27
- Новых находок от адвоката: 5
- Оценок скорректировано адвокатом: 14
