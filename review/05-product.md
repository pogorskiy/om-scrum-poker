# Продуктовый ревьюер — Ревью проекта
- Дата: 2026-04-05
- Статус: done
- Охват: web/src/app.tsx, web/src/main.tsx, web/src/state.ts, web/src/ws.ts, web/index.html, web/src/tokens.css, web/src/global.css, web/src/utils/room-url.ts, web/src/hooks/usePresence.ts, web/src/components/HomePage/HomePage.tsx, web/src/components/HomePage/HomePage.css, web/src/components/RoomPage/RoomPage.tsx, web/src/components/RoomPage/RoomPage.css, web/src/components/Header/Header.tsx, web/src/components/Header/Header.css, web/src/components/Card/Card.tsx, web/src/components/Card/Card.css, web/src/components/CardDeck/CardDeck.tsx, web/src/components/CardDeck/CardDeck.css, web/src/components/ParticipantList/ParticipantList.tsx, web/src/components/ParticipantList/ParticipantList.css, web/src/components/ParticipantCard/ParticipantCard.tsx, web/src/components/ParticipantCard/ParticipantCard.css, web/src/components/NameEntryModal/NameEntryModal.tsx, web/src/components/NameEntryModal/NameEntryModal.css, web/src/components/Toast/Toast.tsx, web/src/components/Toast/Toast.css, web/src/components/ConfirmDialog/ConfirmDialog.tsx, web/src/components/ConfirmDialog/ConfirmDialog.css, internal/server/handler.go, internal/server/ws.go, docs/planning/01-ux-design.md, docs/planning/04-architecture.md
- Модель: claude-opus-4-6

---

## [Доступность / Accessibility]

### [82] Отсутствуют ARIA-атрибуты и поддержка скринридеров для карт голосования
- **Severity:** HIGH
- **Файл:** web/src/components/Card/Card.tsx:19-23, web/src/components/CardDeck/CardDeck.tsx:26-37
- **Проблема:** Карты голосования реализованы как `<button>`, но не имеют ARIA-атрибутов. Нет `role="radio"` или `role="option"`, нет `aria-pressed`/`aria-selected` для обозначения выбранной карты, нет `aria-label` для описания действия. Вся колода не обёрнута в `role="radiogroup"`. Скринридер объявит просто "button 5" без контекста "выбрана карта 5 очков". В документации UX-дизайна (Appendix C) явно указано: "Card selection must be announced by screen readers".
- **Влияние:** Пользователи с нарушениями зрения не смогут эффективно использовать инструмент. Нарушение WCAG 2.1 уровня AA.
- **Рекомендация:** Добавить `aria-pressed={selected}` на каждую карту. Обернуть колоду в `<div role="radiogroup" aria-label="Vote cards">`. Добавить `aria-label` с описательным текстом ("Vote 5 points", "Vote uncertain").
- **Effort:** low

### [78] Отсутствие индикации статуса присутствия для незрячих пользователей
- **Severity:** HIGH
- **Файл:** web/src/components/ParticipantCard/ParticipantCard.tsx:39
- **Проблема:** Статус присутствия (active/idle/disconnected) передаётся исключительно через цвет точки (зелёный/жёлтый/красный). Нет текстового или ARIA-эквивалента. Сам элемент `<span>` не имеет ни `aria-label`, ни `title`, ни скрытого текста. В UX-дизайне (Appendix C) указано: "Presence colors must not be the only indicator — add a text label or icon for colorblind users."
- **Влияние:** Незрячие и дальтоники (до 8% мужчин) не могут определить статус участника.
- **Рекомендация:** Добавить `aria-label={participant.status}` на точку статуса и/или показывать текстовый статус при наведении (tooltip). Для дальтоников можно добавить иконку рядом с точкой.
- **Effort:** low

### ~~[72] Модальные окна не перехватывают фокус клавиатуры (focus trap)~~ ✅ RESOLVED
- **Severity:** HIGH
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.tsx, web/src/components/ConfirmDialog/ConfirmDialog.tsx
- **Проблема:** Ни NameEntryModal, ни ConfirmDialog не реализуют перехват фокуса (focus trap). Пользователь может Tab-ом уйти за пределы модального окна к элементам, которые не должны быть доступны. Также ConfirmDialog не закрывается по Escape (в UX-спеке указано "pressing Escape triggers Cancel"). NameEntryModal не имеет `role="dialog"` и `aria-modal="true"`.
- **Влияние:** Нарушение WCAG 2.1 для модальных диалогов. Пользователи клавиатуры могут потерять фокус и не смогут вернуться в диалог.
- **Рекомендация:** Реализовать focus trap (первый и последний фокусируемый элемент зацикливаются). Добавить обработчик Escape. Добавить `role="dialog"` и `aria-modal="true"`.
- **Effort:** medium
- **Решение:** Все модалки мигрированы на нативный `<dialog>` элемент через shared Modal компонент (`401ce84`). Focus trap, scroll blocking, Escape, ARIA-атрибуты. 41 тест.

### [65] Input-элементы не имеют визуального outline при фокусе
- **Severity:** MEDIUM
- **Файл:** web/src/global.css:34
- **Проблема:** Глобальный стиль `input { outline: none; }` убирает нативный фокусный индикатор у всех полей ввода. Хотя в CSS полей есть `border-color` transition при `:focus`, это не полноценная замена outline — особенно для пользователей, навигирующих клавиатурой. Buttons также не имеют стилей `:focus-visible`.
- **Влияние:** Пользователи клавиатуры не могут чётко видеть, какой элемент в фокусе.
- **Рекомендация:** Добавить `:focus-visible` стили с видимым outline для кнопок и полей ввода. Не убирать outline глобально или заменить его кастомным стилем.
- **Effort:** low

---

## [Обратная связь при ошибках и состояниях соединения]

### [80] Отсутствует визуальный индикатор проблем с соединением
- **Severity:** HIGH
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:58-59, web/src/ws.ts:229-243
- **Проблема:** Когда WebSocket-соединение теряется и идёт переподключение, пользователь видит только страницу "Connecting..." в самом начале. Нет баннера "Reconnecting..." во время работы в комнате. После 30 секунд неудачных попыток показывается toast "Connection lost. Click to retry." — но toast автоматически исчезает через 2.3 секунды (state.ts:137), и пользователь может его пропустить. Кроме того, toast не кликабельный — текст предлагает "Click to retry", но нет обработчика клика. В UX-спеке указан "subtle banner at the top: Reconnecting..." и "Connection lost. [Retry] with a manual retry button".
- **Влияние:** Пользователь может думать, что приложение работает, хотя соединение потеряно. Голоса будут ставиться в очередь и могут устареть. Критично для real-time инструмента.
- **Рекомендация:** Добавить постоянный баннер (не toast) с индикацией состояния соединения: "Reconnecting..." (жёлтый) и "Connection lost" (красный) с кнопкой retry. Toast не подходит для долгосрочных состояний.
- **Effort:** medium

### [62] Комната "Room not found" не реализована
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:49-51, web/src/ws.ts
- **Проблема:** В UX-спеке (9.7) описана страница "Room not found. It may have expired. [Create a New Room]". В реализации есть проверка `if (!roomId)` для невалидного URL, но нет обработки случая, когда комната не существует или истекла на сервере. Если комната была garbage-collected, сервер просто создаст новую — что может быть неожиданно, если пользователь ожидал найти свою старую сессию.
- **Влияние:** Пользователь по старой ссылке попадёт в пустую комнату вместо понятного сообщения об истечении. Низкая, но заметная проблема.
- **Рекомендация:** Это поведение по дизайну (implicit room creation), но стоит хотя бы показать toast "This is a fresh room — previous session may have expired" если пользователь единственный участник.
- **Effort:** low

---

## [Пользовательские флоу и UX]

### [75] Нет возможности сменить имя из интерфейса комнаты
- **Severity:** HIGH
- **Файл:** web/src/components/Header/Header.tsx, web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** В UX-спеке (3.2) указано: "Small pencil icon next to the user's own name on the Room Page allows editing. Changing name broadcasts update to all participants." Также упомянуто меню настроек (gear icon) с опциями "Change my name" и "Leave room". В реализации ни иконки карандаша, ни шестерёнки, ни возможности сменить имя нет. Бэкенд поддерживает `update_name` и фронтенд обрабатывает `name_updated`, но UI для инициации смены имени отсутствует. Единственный способ — очистить localStorage вручную.
- **Влияние:** Пользователь с опечаткой в имени или желающий сменить имя вынужден очищать localStorage — это не доступно обычному пользователю.
- **Рекомендация:** Добавить иконку карандаша рядом с именем в Header или ParticipantCard (для своей карточки), открывающую модалку смены имени.
- **Effort:** low

### [68] Кнопка "Clear Room" доступна любому участнику без ролей
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:138-142
- **Проблема:** Любой участник может очистить комнату, удалив всех. Это по дизайну (документация: "Facilitator role intentionally excluded, all participants are equal"), но кнопка стоит рядом с "New Round" и визуально недостаточно отделена. Случайное нажатие возможно, особенно на мобильных устройствах. Подтверждение есть, но текст "This will remove all participants." может быть недостаточно грозным.
- **Влияние:** Случайная очистка комнаты во время активной сессии — потеря голосов и необходимость всем переподключаться.
- **Рекомендация:** Визуально отделить "Clear Room" от основных действий (например, убрать в меню "..." или сдвинуть в footer). Усилить текст подтверждения.
- **Effort:** low

### [55] Нет визуальной индикации собственного голоса после reveal
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:86-117
- **Проблема:** В UX-спеке (3.3, Reveal Phase) описано: "Your vote: 5" и "(Voting is locked until a new round starts)". В реализации этого нет — пользователь видит свой голос только в общем списке участников. Нет явного указания "ваш голос был 5", и нет сообщения о том, что голосование заблокировано (карты просто неактивны без объяснения).
- **Влияние:** Новый пользователь может не понять, почему карты неактивны и какой голос он отдал.
- **Рекомендация:** Добавить текст "Your vote: X" над заблокированной колодой и пояснение "Voting locked until new round".
- **Effort:** low

### [52] Кнопка "Show Votes" disabled при 0 голосах — отклонение от спеки
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:124-126
- **Проблема:** Кнопка "Show Votes" отключена когда `counts.voted === 0`. В UX-спеке (7.1) явно указано: "Enabled even if 0 votes. The facilitator may want to reveal to prompt discussion." Текущее поведение блокирует этот сценарий.
- **Влияние:** Фасилитатор не может раскрыть голоса для стимуляции обсуждения, если никто ещё не голосовал.
- **Рекомендация:** Убрать `disabled={counts.voted === 0}`. Кнопка должна быть всегда активна в фазе голосования.
- **Effort:** low

### [48] Автоматическая генерация имени комнаты: первый джойнер определяет имя
- **Severity:** MEDIUM
- **Файл:** internal/server/ws.go:155
- **Проблема:** При создании комнаты имя формируется как `p.UserName + "'s Room"`. Если комнату создал один пользователь с именем "Sprint 42 Planning", URL будет содержать slug, но имя комнаты станет "Sprint 42 Planning's Room" — что не совпадает с тем, что пользователь ввёл на домашней странице. Имя комнаты с домашней страницы (`roomName`) вообще не передаётся на сервер — оно используется только для генерации slug в URL.
- **Влияние:** Имя комнаты в Header не соответствует тому, что пользователь вводил. Путаница для пользователей.
- **Рекомендация:** Передавать `roomName` в `join` payload и использовать его при создании комнаты на сервере. Или извлекать из slug (lossy, но приемлемо для MVP).
- **Effort:** medium

---

## [Мобильный опыт]

### [70] Колода карт не зафиксирована внизу экрана на мобильных
- **Severity:** HIGH
- **Файл:** web/src/components/CardDeck/CardDeck.css, web/src/components/RoomPage/RoomPage.css
- **Проблема:** UX-спека (8.4) указывает: "Use position: sticky or position: fixed at the bottom to keep the deck always accessible. The participant area scrolls if there are many participants; the deck does not." В реализации CardDeck — обычный flex/grid контейнер в потоке документа. При большом количестве участников пользователю на мобильном нужно скроллить вниз, чтобы добраться до карт.
- **Влияние:** На мобильных устройствах (заявлено как "mobile-first layout") основной элемент взаимодействия может быть скрыт за скроллом. Особенно критично при 10+ участниках.
- **Рекомендация:** Сделать CardDeck `position: sticky; bottom: 0` с фоном, чтобы карты всегда были видны. Participant area должен скроллиться независимо.
- **Effort:** medium

### [45] Нет оптимизации для landscape-ориентации мобильных
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.css, web/src/components/ParticipantList/ParticipantList.css
- **Проблема:** Нет медиа-запросов для landscape-ориентации. На телефоне в горизонтальном режиме вертикальное пространство сильно ограничено, и участники + кнопки + карты могут не поместиться без скролла.
- **Влияние:** Пользователи на телефонах в landscape (частый кейс при видеозвонке) получат неоптимальный опыт.
- **Рекомендация:** Добавить `@media (orientation: landscape) and (max-height: 500px)` с компактным макетом.
- **Effort:** medium

---

## [Визуальный дизайн и согласованность]

### [58] ~~Расхождение цветовой палитры с UX-спекой~~ ✅ RESOLVED
- **Severity:** MEDIUM
- **Файл:** web/src/tokens.css:6
- **Проблема:** UX-спека (Appendix B) рекомендует Primary: #6366f1 (Indigo). Реализация использует #3b82f6 (Blue). Это не баг — цвет рабочий, но расхождение с проектной документацией.
- **Влияние:** Несоответствие документации и реализации. Может запутать при передаче проекта другому разработчику.
- **Рекомендация:** Выбрать один вариант и обновить либо спеку, либо код.
- **Effort:** low
- **Resolution:** Primary palette updated to Indigo (#6366f1) with hover (#4f46e5), card-selected (#e0e7ff), card-border-selected (#6366f1). Code now matches UX spec.

### [42] ~~Жёстко заданные hex-цвета в hover-стилях~~ ✅ RESOLVED
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.css:43, web/src/components/ConfirmDialog/ConfirmDialog.css:53,61
- **Проблема:** В нескольких местах hover-цвета заданы хардкодом (`#d1d5db`, `#dc2626`) вместо CSS-переменных. Остальная система использует токены (`tokens.css`). Нарушается единообразие, и смена палитры потребует ручного поиска всех значений.
- **Влияние:** Техническая несогласованность. При изменении дизайн-системы можно пропустить эти значения.
- **Рекомендация:** Вынести в CSS-переменные (напр. `--color-border-hover`, `--color-danger-hover`).
- **Effort:** low
- **Resolution:** Extracted 8 new semantic tokens (--color-border-hover, --color-danger, --color-danger-hover, --color-warning-bg/text, --color-danger-bg/text/text-dark). All hardcoded hex colors replaced in RoomPage, ConfirmDialog, EditNameModal, ConnectionBanner CSS files. Verified by automated test scanning all component CSS for hardcoded hex values.

### [35] ~~Нет тёмной темы~~ ✅ RESOLVED
- **Severity:** LOW
- **Файл:** web/src/tokens.css
- **Проблема:** Все цвета определены только для светлой темы. Нет поддержки `prefers-color-scheme: dark`. Многие разработчики (основная аудитория) предпочитают тёмную тему.
- **Влияние:** Белый фон может утомлять глаза при длительных planning-сессиях, особенно в тёмных помещениях.
- **Рекомендация:** Добавить медиа-запрос `@media (prefers-color-scheme: dark)` с альтернативными значениями CSS-переменных. Дизайн-система на токенах это позволяет относительно легко.
- **Effort:** medium
- **Resolution:** Added @media (prefers-color-scheme: dark) block in tokens.css with full coverage of all --color-* and --shadow-* tokens. Dark primary uses Indigo-400 (#818cf8) for contrast. Automated test verifies every light token has a dark override.

---

## [Real-time UX и синхронизация]

### [60] ~~Нет анимации при раскрытии голосов (отсутствие "reveal moment")~~ ✅ RESOLVED
- **Severity:** MEDIUM
- **Файл:** web/src/components/ParticipantCard/ParticipantCard.tsx:24-35
- **Проблема:** Переход от фазы голосования к раскрытию происходит мгновенно — галочки просто заменяются числами. Нет анимации "переворота карт", которая является ключевым моментом scrum poker. UX-спека (2.3) описывает: "All cards flip face-up simultaneously". Отсутствие этой анимации лишает процесс важного визуального момента.
- **Влияние:** Снижается вовлечённость и "геймификация" процесса. Moment of truth теряет драматизм. Пользователи конкурентных инструментов (PlanningPoker.com, Scrum Poker Online) привыкли к этой анимации.
- **Рекомендация:** Добавить CSS-анимацию переворота карты (3D transform rotateY). Уважать `prefers-reduced-motion`.
- **Effort:** medium
- **Resolution:** Added 3D card flip animation (rotateY 180deg) with front/back faces using backface-visibility. Staggered reveal with 80ms delay per card via CSS custom property --flip-delay. Respects prefers-reduced-motion. Covered by 16 unit tests.

### [50] Toast-уведомления исчезают слишком быстро для ошибок
- **Severity:** MEDIUM
- **Файл:** web/src/state.ts:136-138
- **Проблема:** Все toast-ы (и info, и error) исчезают через 2300ms. Для информационных сообщений ("Link copied!") это нормально, но для ошибок ("Connection lost") 2.3 секунды — слишком мало. Пользователь может не успеть прочитать, особенно если он отвлёкся.
- **Влияние:** Пользователь может пропустить важное сообщение об ошибке.
- **Рекомендация:** Увеличить время для error-toast-ов до 5-8 секунд или сделать их persistent (с кнопкой закрытия).
- **Effort:** low

### [40] Нет анимации исчезновения toast-уведомлений
- **Severity:** LOW
- **Файл:** web/src/components/Toast/Toast.css:21-24, web/src/state.ts:136-138
- **Проблема:** UX-спека (4.4) описывает: "2 seconds, then fade out over 300ms". В реализации есть анимация появления (`toast-in`), но нет анимации исчезновения — toast просто удаляется из DOM. Это создаёт рывок.
- **Влияние:** Незначительная, но заметная шероховатость в micro-interactions.
- **Рекомендация:** Добавить fade-out анимацию перед удалением (можно через дополнительный CSS-класс `toast--leaving`).
- **Effort:** low

---

## [Онбординг и первый опыт]

### [56] Домашняя страница не объясняет, что такое scrum poker
- **Severity:** MEDIUM
- **Файл:** web/src/components/HomePage/HomePage.tsx:20-21
- **Проблема:** Подзаголовок "Simple. Self-hosted. No signup." предполагает, что пользователь уже знает, что такое scrum poker. Нет описания для тех, кто получил ссылку от коллеги и не знаком с концепцией. Нет даже подсказки "Planning poker for agile teams" или аналогичного пояснения.
- **Влияние:** Новые пользователи (junior-разработчики, менеджеры) могут быть сбиты с толку.
- **Рекомендация:** Добавить краткое пояснение (1 строка): "Real-time planning poker for agile teams" или аналогичное.
- **Effort:** low

### [44] Нет подсказки по значениям карт для новых пользователей
- **Severity:** MEDIUM
- **Файл:** web/src/components/CardDeck/CardDeck.tsx:6
- **Проблема:** Колода `['?', '0', '0.5', '1', '2', '3', '5', '8', '13', '20', '40', '100']` показывается без пояснений. Новый пользователь может не понимать, что означает "?", почему последовательность нелинейная (Fibonacci-like) и что значат конкретные числа.
- **Влияние:** Пользователи без опыта в scrum poker выберут произвольное число вместо осознанной оценки.
- **Рекомендация:** Добавить tooltip на "?" ("I'm not sure / Need more discussion"). Опционально — краткую подсказку при первом визите.
- **Effort:** low

---

## [Масштабируемость UX]

### [46] Список участников не скроллится при большом количестве
- **Severity:** MEDIUM
- **Файл:** web/src/components/ParticipantList/ParticipantList.css
- **Проблема:** ParticipantList — grid с `auto-fill`, растущий без ограничений. При 20+ участниках список займёт весь экран, оттеснив кнопки действий и колоду карт за видимую область. Нет `max-height` или `overflow-y: auto`.
- **Влияние:** При больших сессиях (10-20 участников) UX деградирует, особенно на мобильных.
- **Рекомендация:** Ограничить высоту ParticipantList (напр. `max-height: 40vh; overflow-y: auto`) или использовать sticky CardDeck (см. находку выше).
- **Effort:** low

### [38] Нет индикации "все проголосовали" кроме текста кнопки
- **Severity:** LOW
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:127
- **Проблема:** Когда все участники проголосовали, единственная индикация — текст кнопки "Show Votes (5 of 5 voted)". Нет визуального выделения, звука или подсветки. Конкуренты обычно подсвечивают кнопку или показывают баннер "Everyone has voted!".
- **Влияние:** Фасилитатор может не заметить, что все проголосовали, и затянет ожидание.
- **Рекомендация:** Подсветить кнопку (пульсация, смена цвета) когда voted === total. Опционально — toast "Everyone voted!".
- **Effort:** low

---

## [Отсутствующие стандартные функции]

### [54] Нет возможности "наблюдатель" (observer / spectator mode)
- **Severity:** MEDIUM
- **Файл:** web/src/components/RoomPage/RoomPage.tsx, internal/server/ws.go
- **Проблема:** В большинстве scrum poker инструментов есть роль "наблюдатель" — Scrum Master или Product Owner, который не голосует, но управляет процессом. В om-scrum-poker все участники являются голосующими. Это означает, что SM будет отображаться как "не проголосовал", путая счётчик голосов.
- **Влияние:** Команды, где SM не голосует, вынуждены игнорировать его в счётчике — визуальный шум.
- **Рекомендация:** Добавить опцию "Join as observer" в NameEntryModal. Наблюдатели отображаются отдельно и не учитываются в счётчике голосов.
- **Effort:** high

### [47] Нет истории раундов в рамках сессии
- **Severity:** MEDIUM
- **Файл:** web/src/ws.ts:166-179
- **Проблема:** При нажатии "New Round" все результаты предыдущего раунда теряются безвозвратно. На типичной planning-сессии команда оценивает 10-20 задач. Без истории невозможно вернуться к результатам предыдущей оценки. Документация осознанно исключает серверную историю, но даже клиентская in-memory история раундов (для текущей сессии) была бы полезна.
- **Влияние:** Команды вынуждены вручную записывать результаты каждого раунда (в Jira, на бумаге и т.д.).
- **Рекомендация:** Хранить историю раундов в state (клиентская, in-memory). Показывать простой список "Round 1: Average 5, Median 5 / Round 2: Average 8, Median 8" в свёрнутой панели.
- **Effort:** medium

### [32] Нет кнопки "Leave room"
- **Severity:** LOW
- **Файл:** web/src/components/Header/Header.tsx
- **Проблема:** UX-спека описывает меню настроек (gear icon) с опцией "Leave room". В реализации нет такой кнопки. Пользователь может покинуть комнату только закрыв вкладку или навигировав на другую страницу. При этом бэкенд поддерживает событие `leave`.
- **Влияние:** Неполный UI. Пользователь остаётся "disconnected" в списке вместо чистого ухода.
- **Рекомендация:** Добавить кнопку "Leave" в Header (рядом с "Copy Link" или в dropdown-меню).
- **Effort:** low

### [28] Нет поддержки ссылки на конкретную задачу (topic/story)
- **Severity:** LOW
- **Файл:** web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** В большинстве scrum poker инструментов можно указать тему текущего раунда (номер задачи, название user story). В om-scrum-poker раунды анонимны — нет контекста, что сейчас оценивается. Это осознанный выбор (KISS), но даже простое текстовое поле "Current topic" было бы полезно.
- **Влияние:** При записи результатов команда должна отдельно отслеживать, какой раунд к какой задаче относится.
- **Рекомендация:** Добавить опциональное поле "Topic" (фасилитатор вводит название задачи), синхронизируемое через WS. Минимально: просто readonly-текст вверху страницы.
- **Effort:** medium

---

## [Безопасность UX]

### [43] Нет валидации input на домашней странице
- **Severity:** MEDIUM
- **Файл:** web/src/components/HomePage/HomePage.tsx:28-30, web/src/utils/room-url.ts:3-10
- **Проблема:** UX-спека (3.1) указывает: "Validation: Alphanumeric, spaces, hyphens, underscores only. Inline error for invalid characters." Реализация не валидирует input — slugify просто удаляет невалидные символы. Пользователь может ввести "!!!" и получить пустой slug (только hex-часть). Нет inline-ошибки, нет визуальной обратной связи.
- **Влияние:** Пользователь может создать комнату с нечитаемым URL. Не критично, но снижает UX.
- **Рекомендация:** Добавить валидацию с inline-сообщением для невалидных символов. Или показывать превью URL под полем ввода.
- **Effort:** low

### [30] NameEntryModal позволяет ввести имя из одних пробелов (визуально)
- **Severity:** LOW
- **Файл:** web/src/components/NameEntryModal/NameEntryModal.tsx:11-12
- **Проблема:** Кнопка "Join" дизейблится проверкой `!name.trim()`, но нет inline-сообщения и нет ограничения на минимальную длину (кроме 1 символа). Имя "A" технически валидно. Также нет визуальной обратной связи при попытке отправки пустого имени.
- **Влияние:** Минимальная — trim предотвращает основной случай.
- **Рекомендация:** Для полноты можно добавить минимальную длину 2 символа и inline-подсказку.
- **Effort:** low

---

## [Документация для пользователей]

### [35] Нет пользовательской документации / help
- **Severity:** LOW
- **Файл:** (отсутствует)
- **Проблема:** Нет страницы помощи, FAQ, или хотя бы tooltips объясняющих функциональность. README содержит только техническую информацию для разработчиков. Новый пользователь, получивший ссылку, должен разобраться самостоятельно.
- **Влияние:** Увеличивается порог входа для нетехнических пользователей (PO, аналитики).
- **Рекомендация:** Добавить минимальный "How it works" блок на домашней странице или tooltip-подсказки при первом визите.
- **Effort:** low

---

## [Технические UX-детали]

### [25] HTML title не обновляется при входе в комнату
- **Severity:** LOW
- **Файл:** web/index.html:5, web/src/components/RoomPage/RoomPage.tsx
- **Проблема:** Title всегда "om-scrum-poker". При работе в комнате было бы полезно видеть имя комнаты во вкладке браузера (например, "Sprint 42 — om-scrum-poker"). Это помогает при работе с несколькими вкладками.
- **Влияние:** Косметическая проблема, но влияет на UX при мультитаскинге.
- **Рекомендация:** Обновлять `document.title` при получении `room_state`.
- **Effort:** low

### [22] Нет favicon, кроме emoji
- **Severity:** LOW
- **Файл:** web/index.html:7
- **Проблема:** Favicon реализован как inline SVG с emoji. Это работает, но не во всех браузерах отображается одинаково. Нет manifest.json для PWA.
- **Влияние:** Минимальная. Выглядит менее профессионально.
- **Рекомендация:** Добавить полноценный favicon (SVG + PNG fallback) и web manifest.
- **Effort:** low

### [18] Нет meta description и Open Graph тегов
- **Severity:** NOTE
- **Файл:** web/index.html
- **Проблема:** Нет `<meta name="description">` и Open Graph тегов. При публикации ссылки в Slack/Teams предпросмотр будет пустым.
- **Влияние:** Ссылка на домашнюю страницу в мессенджерах выглядит невзрачно.
- **Рекомендация:** Добавить meta description и основные OG-теги (title, description, type).
- **Effort:** low

### [15] Нет prefers-reduced-motion
- **Severity:** NOTE
- **Файл:** web/src/components/Card/Card.css:16, web/src/components/Toast/Toast.css:30-39
- **Проблема:** UX-спека (Appendix C) указывает: "The reveal animation should respect prefers-reduced-motion." Текущие CSS-анимации (card transform, toast-in) не учитывают это предпочтение.
- **Влияние:** Пользователи с вестибулярными расстройствами могут испытывать дискомфорт.
- **Рекомендация:** Добавить `@media (prefers-reduced-motion: reduce) { * { animation: none; transition: none; } }` или точечно отключить анимации.
- **Effort:** low

### [12] Нет индикации высокого spread при раскрытии голосов
- **Severity:** NOTE
- **Файл:** web/src/components/RoomPage/RoomPage.tsx:86-117
- **Проблема:** UX-спека (10.3) рекомендует: "High spread (3 to 13) — discussion recommended" с нейтральным информационным стилем. Данные `spread` приходят в `VoteResult`, но не отображаются. Consensus отображается, а высокий spread — нет.
- **Влияние:** Команда может пропустить момент, когда оценки сильно расходятся и нужно обсуждение.
- **Рекомендация:** Добавить визуальную индикацию spread: диапазон значений и рекомендацию обсудить при высоком расхождении.
- **Effort:** low

---

## Саммари
- Всего находок: 27
- CRITICAL (90-100): 0
- HIGH (70-89): 5
- MEDIUM (40-69): 13
- LOW (20-39): 6
- NOTE (0-19): 3
- Средний балл: 47.6
- Топ-3 проблемы:
  1. [82] Отсутствуют ARIA-атрибуты и поддержка скринридеров для карт голосования
  2. [80] Отсутствует визуальный индикатор проблем с соединением
  3. [78] Отсутствие индикации статуса присутствия для незрячих пользователей
