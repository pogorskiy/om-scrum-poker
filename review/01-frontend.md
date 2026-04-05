# Ведущий фронтенд-разработчик — Ревью проекта
- Дата: 2026-04-05
- Статус: done
- Охват: web/index.html, web/package.json, web/tsconfig.json, web/vite.config.ts, web/src/main.tsx, web/src/app.tsx, web/src/state.ts, web/src/ws.ts, web/src/global.css, web/src/tokens.css, web/src/hooks/usePresence.ts, web/src/utils/room-url.ts, web/src/components/Card/Card.tsx+css, web/src/components/CardDeck/CardDeck.tsx+css, web/src/components/Header/Header.tsx+css, web/src/components/HomePage/HomePage.tsx+css, web/src/components/NameEntryModal/NameEntryModal.tsx+css, web/src/components/ParticipantCard/ParticipantCard.tsx+css, web/src/components/ParticipantList/ParticipantList.tsx+css, web/src/components/RoomPage/RoomPage.tsx+css, web/src/components/Toast/Toast.tsx+css, web/src/components/ConfirmDialog/ConfirmDialog.tsx+css
- Модель: claude-opus-4-6

---

## Accessibility (a11y)

### ~~[82] Модальные окна не управляют фокусом и не блокируют скролл~~ ✅ RESOLVED

- **Severity:** HIGH
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.tsx, web/src/components/ConfirmDialog/ConfirmDialog.tsx
- **Проблема:** Модальные окна (NameEntryModal, ConfirmDialog) не используют focus trap. Пользователь может табом выйти за пределы модалки и взаимодействовать с элементами под оверлеем. Нет `role="dialog"`, `aria-modal="true"`, `aria-labelledby`. Нет закрытия по Escape. Нет блокировки скролла body.
- **Влияние:** Пользователи с клавиатурной навигацией и скринридерами не смогут нормально использовать модалки. Нарушение WCAG 2.1 AA (2.4.3 Focus Order, 4.1.2 Name/Role/Value).
- **Рекомендация:** Добавить focus trap (можно на `<dialog>` элементе или вручную), `role="dialog"`, `aria-modal="true"`, `aria-labelledby`, обработку Escape, блокировку scroll на body.
- **Effort:** medium
- **Решение:** Все модалки мигрированы на нативный `<dialog>` элемент через shared Modal компонент (`401ce84`). Focus trap, scroll blocking, Escape, backdrop click, ARIA-атрибуты, focus restore. 41 тест.

### [72] Отсутствуют ARIA-атрибуты на интерактивных элементах

- **Severity:** HIGH
- **Файл:** web/src/components/Card/Card.tsx, web/src/components/ParticipantCard/ParticipantCard.tsx, web/src/components/Header/Header.tsx
- **Проблема:** Карточки голосования не имеют `aria-pressed` для отображения состояния выбора. Статус-индикаторы (цветные точки active/idle/disconnected) не имеют `aria-label` — скринридер их не прочитает. Кнопка "Copy Link" не имеет обратной связи через `aria-live`.
- **Влияние:** Пользователи скринридеров не получают информацию о состоянии голосования и статусе участников.
- **Рекомендация:** Добавить `aria-pressed={selected}` на Card, `aria-label` на статус-индикаторы, использовать `aria-live="polite"` для toast-уведомлений.
- **Effort:** low

### [55] Toast-уведомления недоступны для скринридеров

- **Severity:** MEDIUM
- **Файл:** web/src/components/Toast/Toast.tsx
- **Проблема:** Toast-контейнер не имеет `role="status"` или `aria-live="polite"`. Скринридеры не озвучат появление уведомлений.
- **Влияние:** Незрячие пользователи пропустят все уведомления (join/leave, ошибки, копирование ссылки).
- **Рекомендация:** Добавить `role="status"` и `aria-live="polite"` на контейнер, `role="alert"` для error-тоастов.
- **Effort:** low

---

## WebSocket и сетевое взаимодействие

### [75] Очередь сообщений не ограничена по размеру

- **Severity:** HIGH
- **Файл:** web/src/ws.ts:18
- **Проблема:** `messageQueue` массив растёт неограниченно, пока WebSocket отключён. Если пользователь активно кликает во время disconnect (например, повторно голосует), очередь будет копить сообщения. После reconnect все сообщения отправятся разом, что может привести к неожиданному поведению (устаревшие голоса, конфликтующие команды).
- **Влияние:** Потенциально некорректное состояние после реконнекта; на практике маловероятно из-за блокировки UI, но архитектурно опасно.
- **Рекомендация:** Ограничить размер очереди (например, 20 сообщений), при reconnect очищать очередь и полагаться на `room_state` для восстановления. Или вообще не ставить в очередь vote/reveal/reset — они теряют смысл при reconnect.
- **Effort:** low

### [62] "Connection lost. Click to retry." — но нет обработчика клика

- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:233
- **Проблема:** После 30-секундного таймаута реконнекта показывается toast "Connection lost. Click to retry.", но toast — это пассивный элемент с `pointer-events: auto`, без обработчика клика. Функция `retry()` экспортирована, но нигде не используется.
- **Влияние:** Пользователь видит инструкцию "Click to retry", но клик ничего не делает. Единственный выход — перезагрузка страницы.
- **Рекомендация:** Либо добавить кликабельный retry в UI (отдельный компонент или кликабельный toast), либо изменить текст на "Connection lost. Please reload the page."
- **Effort:** low

### [45] Нет валидации входящих WebSocket-сообщений

- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:55-62
- **Проблема:** `JSON.parse` оборачивается в try/catch (хорошо), но результат кастуется через `as ServerMessage` без runtime-проверки. Если сервер пришлёт неожиданную структуру, `switch` просто не сматчит (неплохо), но внутри case-блоков обращение к `msg.payload.sessionId` и т.п. может вызвать runtime error при некорректной структуре payload.
- **Влияние:** Маловероятно при контролируемом сервере, но нарушает принцип defense in depth.
- **Рекомендация:** Добавить минимальную проверку наличия `type` и `payload` перед switch, или использовать assertion-функции.
- **Effort:** low

---

## Управление состоянием

### [58] Мутация roomState вместо иммутабельного обновления в room_cleared

- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:184-185
- **Проблема:** В обработчике `room_cleared` сначала `roomState.value = null`, затем сразу `send(join)`. Между этими строками компоненты, зависящие от `roomState`, рендерятся с null, что может вызвать мерцание UI (ParticipantList вернёт null, RoomPage покажет "Connecting..."). Сигнал обновляется синхронно, что вызывает промежуточный рендер.
- **Влияние:** Визуальное мерцание при очистке комнаты — пользователь видит кратковременное "Connecting...".
- **Рекомендация:** Использовать `batch()` из `@preact/signals` или не обнулять `roomState`, а ждать нового `room_state` от сервера после re-join.
- **Effort:** low

### [42] selectedCard не сбрасывается при смене комнаты через URL

- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:34-41
- **Проблема:** `selectedCard.value = ''` сбрасывается только в cleanup `useEffect`. Если пользователь сначала заходит в комнату A, голосует, затем через URL переходит в комнату B — cleanup сработает, но между disconnect и новым connect может быть состояние гонки.
- **Влияние:** Минимальное — cleanup правильно очищает при unmount. Но при изменении roomId без unmount (если path изменится с `/room/a` на `/room/b`) порядок cleanup/setup может зависеть от фреймворка.
- **Рекомендация:** Явный сброс `selectedCard.value = ''` в начале connect() функции.
- **Effort:** low

---

## Безопасность

### [68] XSS через userName в toast-уведомлениях

- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:103, web/src/ws.ts:121
- **Проблема:** `addToast(\`${msg.payload.userName} joined\`)` — имя пользователя от сервера подставляется в сообщение. Хотя Preact экранирует строки при рендере (защита от XSS в DOM), если в будущем toast станет рендерить HTML (dangerouslySetInnerHTML), это станет вектором атаки. Кроме того, userName не санитизируется на клиенте — можно ввести имя в сотни символов.
- **Влияние:** Текущий риск низкий благодаря Preact-экранированию. Но длинные имена могут сломать layout.
- **Рекомендация:** Ограничить отображение userName до N символов (truncate), полагаться на серверную валидацию длины.
- **Effort:** low

### [35] Session ID хранится в localStorage без привязки к устройству

- **Severity:** LOW
- **Файл:** web/src/state.ts:88-94
- **Проблема:** Session ID генерируется один раз и живёт вечно в localStorage. Если кто-то получит доступ к localStorage (XSS, shared computer), он может использовать чужой session ID. Нет механизма ротации или инвалидации.
- **Влияние:** При данной threat model (нет авторизации, in-memory state) риск минимален — session ID теряет смысл после перезапуска сервера.
- **Рекомендация:** Для текущего масштаба — acceptable risk. При добавлении авторизации — пересмотреть.
- **Effort:** low

---

## Производительность

### [48] Неоптимальное обновление массива participants на каждое событие

- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:87-220
- **Проблема:** Каждое событие (vote_cast, presence_changed, name_updated и др.) создаёт новый объект `roomState` через spread + map. Это вызывает полный ре-рендер всех компонентов, зависящих от `roomState`. При 20+ участниках, активно голосующих, это может быть заметно.
- **Влияние:** В типичном сценарии (5-10 участников) — незаметно. При 20+ — потенциальные фризы на слабых устройствах.
- **Рекомендация:** Для текущего масштаба приемлемо. При росте — рассмотреть отдельные сигналы для participants или использовать `@preact/signals` computed selectors для точечных обновлений.
- **Effort:** high

### [25] Отсутствие lazy loading и code splitting

- **Severity:** LOW
- **Файл:** web/vite.config.ts
- **Проблема:** Всё приложение загружается одним бандлом. HomePage и RoomPage не разделены.
- **Влияние:** При текущем размере приложения (~30 файлов, минимум зависимостей) это не проблема. Preact + signals — маленькие библиотеки.
- **Рекомендация:** Не требуется действий при текущем размере. Учитывать при значительном росте.
- **Effort:** medium

---

## UX и обработка ошибок

### [65] Нет индикации состояния подключения в UI

- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** `connectionStatus` сигнал используется только для показа "Connecting..." при первом подключении. При потере связи и реконнекте пользователь не видит никакого индикатора — UI выглядит нормально, но действия не отправляются (ставятся в очередь). Единственная обратная связь — toast через 30 секунд при полном отключении.
- **Влияние:** Пользователь может голосовать, думая что голос отправлен, хотя WebSocket разорван.
- **Рекомендация:** Добавить визуальный индикатор статуса соединения (цветная точка в Header, баннер "Reconnecting..." при disconnected/connecting).
- **Effort:** low

### [60] Нет защиты от множественного вызова reveal/new_round

- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:62-68
- **Проблема:** Кнопки "Show Votes" и "New Round" не дизейблятся после клика. Пользователь может быстро кликнуть несколько раз, отправив дублирующие команды серверу.
- **Влияние:** Зависит от серверной идемпотентности. Скорее всего, сервер обработает повторы без проблем, но это лишний трафик.
- **Рекомендация:** Добавить debounce или блокировку кнопки после первого клика до получения ответа от сервера.
- **Effort:** low

### [52] Кнопка "Clear Room" доступна всем участникам без различия ролей

- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:138-142
- **Проблема:** Любой участник может очистить комнату, удалив всех остальных. ConfirmDialog — слабая защита, так как это всего один клик. Нет концепции "владельца комнаты".
- **Влияние:** Любой участник может случайно или злонамеренно очистить комнату.
- **Рекомендация:** На текущем этапе — acceptable, так как проект позиционируется как "no auth, KISS". При росте пользовательской базы — добавить роль owner/admin.
- **Effort:** high

---

## Код и архитектура

### [38] Дублирование логики маппинга vote в participants

- **Severity:** LOW
- **Файл:** web/src/ws.ts:66-74, web/src/ws.ts:151-159
- **Проблема:** Код маппинга votes на participants (создание votesMap из result.votes, затем map participants с добавлением vote) дублируется в обработчиках `room_state` и `votes_revealed`.
- **Влияние:** Нарушение DRY. При изменении логики нужно менять в двух местах.
- **Рекомендация:** Вынести в отдельную функцию `applyVotesToParticipants(participants, votes)`.
- **Effort:** low

### [32] Хардкод значений карточек

- **Severity:** LOW
- **Файл:** web/src/components/CardDeck/CardDeck.tsx:6
- **Проблема:** `CARD_VALUES` захардкожены на клиенте. Если сервер поддерживает другие наборы (Fibonacci, T-shirt sizes), клиент не сможет их отобразить.
- **Влияние:** Ограничивает расширяемость. На текущем этапе не проблема.
- **Рекомендация:** Рассмотреть получение допустимых значений от сервера в `room_state`.
- **Effort:** medium

### [28] Отсутствие тестов фронтенда

- **Severity:** LOW
- **Файл:** web/package.json
- **Проблема:** Нет ни одного теста. Нет test-зависимостей (vitest, @testing-library/preact). Критическая логика (ws.ts, state.ts, room-url.ts) не покрыта.
- **Влияние:** Регрессии при изменениях невозможно отловить автоматически.
- **Рекомендация:** Добавить vitest + @testing-library/preact. Начать с юнит-тестов ws.ts (handleMessage), state.ts (computed signals), room-url.ts (parseRoomId, generateRoomUrl).
- **Effort:** medium

### [22] usePresence добавляет 4 глобальных обработчика событий

- **Severity:** LOW
- **Файл:** web/src/hooks/usePresence.ts:40
- **Проблема:** mousemove, keydown, touchstart, scroll — все вешаются на window. mousemove срабатывает очень часто. Хотя функция `resetIdleTimer` лёгкая (проверка ref + clearTimeout + setTimeout), при быстром движении мыши это лишняя нагрузка.
- **Влияние:** На современных устройствах незаметно. На мобильных устройствах с touchstart/scroll — приемлемо (passive: true стоит).
- **Рекомендация:** Можно добавить throttle на resetIdleTimer (например, 1 раз в секунду). Но при текущей простоте функции — optional.
- **Effort:** low

---

## CSS и стилизация

### [30] Хардкод цвета #d1d5db в нескольких местах

- **Severity:** LOW
- **Файл:** web/src/components/RoomPage/RoomPage.css:41, web/src/components/ConfirmDialog/ConfirmDialog.css:52
- **Проблема:** `background: #d1d5db` используется в hover-состояниях secondary кнопок, но не вынесен в CSS-переменную в tokens.css. Аналогично `#dc2626` в ConfirmDialog.
- **Влияние:** Непоследовательность дизайн-системы. При изменении палитры легко пропустить.
- **Рекомендация:** Добавить `--color-border-hover` и `--color-danger-hover` в tokens.css.
- **Effort:** low

### [18] Нет тёмной темы

- **Severity:** NOTE
- **Файл:** web/src/tokens.css
- **Проблема:** Все цвета в :root. Нет `@media (prefers-color-scheme: dark)` или переключателя темы.
- **Влияние:** Пользователи с тёмной системной темой видят белый интерфейс.
- **Рекомендация:** Добавить тёмную тему через CSS-переменные в media query. Дизайн-токены уже вынесены — адаптация будет простой.
- **Effort:** medium

### [15] Нет анимации исчезновения toast

- **Severity:** NOTE
- **Файл:** web/src/components/Toast/Toast.css
- **Проблема:** Toast появляется с анимацией `toast-in`, но исчезает мгновенно (элемент просто удаляется из DOM через setTimeout в state.ts).
- **Влияние:** Визуальный скачок при исчезновении. Минорный UX-момент.
- **Рекомендация:** Добавить fade-out анимацию перед удалением (например, через CSS-класс + transitionend).
- **Effort:** low

---

## Конфигурация сборки

### [12] tsconfig: target ES2020 при доступном ES2022+

- **Severity:** NOTE
- **Файл:** web/tsconfig.json
- **Проблема:** Target ES2020. В 2026 году все современные браузеры поддерживают ES2022+. Это не влияет на итоговый бандл (Vite использует esbuild), но влияет на type-checking.
- **Влияние:** Нет практического влияния при текущих зависимостях.
- **Рекомендация:** Обновить до ES2022 для доступа к `Array.at()`, `Object.hasOwn()` и другим API в типах.
- **Effort:** low

### [10] index.html: нет meta description и Open Graph

- **Severity:** NOTE
- **Файл:** web/index.html
- **Проблема:** Минимальный `<head>` — только charset, viewport и title. Нет description, OG-тегов, favicon в нескольких размерах.
- **Влияние:** При шаринге ссылки в мессенджерах не будет превью. SEO не актуально для self-hosted инструмента.
- **Рекомендация:** Добавить минимум `<meta name="description">` и `<meta property="og:title">` для красивого превью при шаринге.
- **Effort:** low

---

## Положительные аспекты

- **Минимальные зависимости:** Только preact + signals, никакого bloat. Идеально для KISS-проекта.
- **Чистая архитектура:** Состояние отделено от компонентов, WS-логика изолирована, компоненты атомарные.
- **Дизайн-токены:** Вся палитра, spacing, shadows через CSS-переменные.
- **BEM-нейминг:** Последовательно применяется во всех компонентах.
- **TypeScript strict mode:** Включен, типы корректные, discriminated union для сообщений.
- **Immutable state updates:** Все обновления через spread (кроме room_cleared).
- **Reconnect с exponential backoff + jitter:** Правильная реализация переподключения.
- **Graceful localStorage handling:** try/catch для private browsing.
- **Mobile-first grid:** Адаптивные layouts через CSS grid с медиа-запросами.

---

## Саммари
- Всего находок: 21
- CRITICAL (90-100): 0
- HIGH (70-89): 3
- MEDIUM (40-69): 8
- LOW (20-39): 6
- NOTE (0-19): 4
- Средний балл: 43.7
- Топ-3 проблемы:
  1. ~~[82] Модальные окна не управляют фокусом (a11y)~~ ✅ RESOLVED
  2. [75] Очередь WebSocket-сообщений не ограничена
  3. [72] Отсутствуют ARIA-атрибуты на интерактивных элементах
