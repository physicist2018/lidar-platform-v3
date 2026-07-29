# Frontend Development Guide

## Architecture

Фронтенд — SPA на Vite + vanilla JavaScript (без фреймворков). Состоит из:

```
frontend/
├── index.html          — точка входа, навигация, Plotly CDN
├── vite.config.js      — прокси API через nginx
├── package.json        — зависимости (Vite, Plotly)
│
└── src/
    ├── main.js         — корневой роутер, регистрация страниц
    ├── api.js          — HTTP-клиент (JWT, 401 → login)
    ├── router.js       — hash-based SPA роутер
    ├── store.js        — глобальное состояние (pub/sub)
    ├── styles.css      — UI kit (CSS-переменные, карточки, таблицы, формы)
    │
    └── pages/          — каждая страница — отдельный модуль
        ├── login.js
        ├── register.js
        ├── verified.js
        ├── experiments.js
        ├── experiment-detail.js
        ├── upload.js
        └── prepared.js
```

**Ключевые принципы:**
- Нет фреймворка — только vanilla JS
- Страницы — функции, возвращающие cleanup
- Состояние — простой объект с pub/sub
- API-слой — единый модуль с автоподстановкой JWT

---

## Добавление новой страницы

### 1. Создать файл страницы

`frontend/src/pages/my-page.js`:

```js
import { getState, setState } from "../store.js";
import { someApiFunction } from "../api.js";

/**
 * renderMyPage — функция страницы.
 * @param {HTMLElement} container — корневой DOM-элемент для страницы
 * @param {Object} params — параметры из URL (напр. { id: "uuid" } для /page/:id)
 * @returns {Function|null} cleanup-функция (если есть таймеры/подписки)
 */
export function renderMyPage(container, params) {
  // 1. Рендерим HTML
  container.innerHTML = `
    <div class="card">
      <h1>My Page</h1>
      <p>ID: ${params?.id || "—"}</p>
      <button class="btn btn-primary" id="my-btn">Click me</button>
      <div id="my-error" class="flash flash-error" hidden></div>
    </div>
  `;

  // 2. Навешиваем обработчики
  const btn = document.getElementById("my-btn");
  const errorEl = document.getElementById("my-error");

  btn.addEventListener("click", async () => {
    try {
      const data = await someApiFunction();
      // обновить UI
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    }
  });

  // 3. Возвращаем cleanup (опционально)
  const interval = setInterval(() => { /* refresh */ }, 5000);
  return () => clearInterval(interval);
}
```

### 2. Зарегистрировать роут

В `frontend/src/main.js`:

```js
import { renderMyPage } from "./pages/my-page.js";

route("/my-page", renderMyPage);
route("/my-page/:id", renderMyPage);  // с динамическим параметром
```

### 3. Добавить ссылку в навигацию

В `frontend/index.html`:

```html
<a href="#/my-page">My Page</a>
```

---

## Добавление API-функции

В `frontend/src/api.js`:

```js
// ---------------------------------------------------------------------------
// My Feature API
// ---------------------------------------------------------------------------

export function getMyData(params) {
  return request("GET", "/api/v1/my-endpoint", { params });
}

export function createMyItem(body) {
  return request("POST", "/api/v1/my-endpoint", { body });
}
```

**request(method, path, options) — основной HTTP-клиент:**

| Опция | Тип | Назначение |
|---|---|---|
| `body` | `Object` | JSON-тело запроса (автоматически `JSON.stringify`) |
| `formData` | `FormData` | Multipart-форма (Content-Type не задаётся) |
| `params` | `Object` | Query-параметры (преобразуются в URLSearchParams) |

**Автоматически:**
- Добавляет `Authorization: Bearer <token>` (берёт из localStorage)
- При 401 — очищает токен и редиректит на `/login`
- Парсит JSON-ответ, при ошибке выбрасывает `Error` с полем `error`

---

## Роутер

Роутер — hash-based: `#/path`.

### Статические роуты

```js
route("/login", renderLogin);      // #/login
route("/experiments", renderExperiments);  // #/experiments
```

### Динамические роуты (`:param`)

```js
route("/experiments/:id", renderExperimentDetail);  // #/experiments/uuid
```

В `params` функции страницы: `params.id === "uuid"`.

### Вложенные роуты

Не поддерживаются на уровне роутера. Для разных состояний одной страницы
используйте query-параметры или вложенные секции внутри страницы.

### Cleanup

Если страница создаёт интервалы или подписки — `render*` функция должна
вернуть cleanup-функцию. Роутер вызывает её при уходе со страницы.

```js
export function renderPage(container) {
  const interval = setInterval(refresh, 5000);
  // Cleanup возвращается синхронно (не async!)
  return () => clearInterval(interval);
}
```

**Важно:** если функция `async`, роутер не сможет получить cleanup
(возвращается Promise). Используйте модульные переменные для очистки:

```js
let intervalId = null;

export async function renderPage(container) {
  if (intervalId) clearInterval(intervalId);
  // ...
  intervalId = setInterval(refresh, 5000);
}
```

---

## Состояние (store)

Глобальное состояние — простой объект. Подписка на изменения через pub/sub.

```js
import { getState, setState, subscribe } from "../store.js";

// Чтение
const { token, experiments } = getState();

// Запись (слияние с текущим состоянием)
setState({ currentExperiment: exp });

// Подписка на изменения
const unsub = subscribe((state) => {
  console.log("state changed", state);
});
// Отписка
unsub();
```

**Текущие поля состояния:**

| Поле | Тип | Назначение |
|---|---|---|
| `token` | `string\|null` | JWT-токен из localStorage |
| `experiments` | `array` | Список экспериментов |
| `currentExperiment` | `object\|null` | Текущий выбранный эксперимент |

---

## Стили

Единый `styles.css` с CSS-переменными. Не требует CSS-фреймворков.

### CSS-переменные

```css
--color-primary: #3b82f6;      /* синий — основные кнопки */
--color-danger: #ef4444;       /* красный — ошибки, удаление */
--color-success: #22c55e;      /* зелёный — успех */
--color-bg: #f5f7fa;           /* фон страницы */
--color-surface: #ffffff;      /* фон карточек */
--color-text: #1e293b;         /* основной текст */
--color-text-muted: #64748b;   /* второстепенный текст */
--color-border: #e2e8f0;       /* границы */
--radius: 8px;                 /* скругление */
--shadow: 0 1px 3px rgba(0,0,0,0.08); /* тень карточек */
--max-width: 960px;            /* максимальная ширина контента */
```

### Основные классы

| Класс | Назначение |
|---|---|
| `.card` | Контейнер с тенью и скруглением |
| `.btn` | Кнопка |
| `.btn-primary` | Основная кнопка (синяя) |
| `.btn-outline` | Кнопка с обводкой |
| `.form-group` | Группа "label + input" |
| `.form-row` | Ряд полей (flex, gap) |
| `.form-actions` | Ряд кнопок |
| `.flash-error` | Красное сообщение об ошибке |
| `.flash-success` | Зелёное сообщение об успехе |
| `.flash-info` | Синее информационное сообщение |
| `.table-wrap` | Обёртка для скролла таблицы |
| `.badge` | Статусный тег |
| `.badge-pending` | Жёлтый — "ожидает" |
| `.badge-processing` | Синий — "в работе" |
| `.badge-completed` | Зелёный — "готово" |
| `.badge-failed` | Красный — "ошибка" |
| `.loading` | Центрированный спиннер-заглушка |
| `.empty-state` | Центрированное сообщение "нет данных" |
| `.detail-grid` | Сетка для деталей эксперимента |
| `.detail-item` | Элемент с dt/dd |

**Важно:** не нужно добавлять CSS-фреймворки. Все стили уже есть.

---

## Паттерны

### Форма с загрузкой файла (multipart)

```js
const form = document.getElementById("my-form");
form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  // или fd.append("key", value) для дополнительных полей
  const result = await createExperiment(fd);
});
```

### Форма с JSON (создание задачи)

```js
form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const payload = {
    experiment_id: expId,
    background_type: "mean",
  };
  const result = await createTask({
    subject: "lidar.task.prepare_experiment",
    task_type: "prepare_experiment",
    payload,
  });
});
```

### Каскадные селекты (зависимые выпадающие списки)

Паттерн: при изменении родительского селекта → очищаем дочерние → загружаем данные → заполняем.

```js
parentSelect.addEventListener("change", async () => {
  childSelect.innerHTML = `<option value="">— загрузка —</option>`;

  const data = await fetchData(parentSelect.value);

  childSelect.innerHTML =
    `<option value="">— все —</option>` +
    data.items.map((item) => `<option value="${item}">${item}</option>`).join("");
});
```

### График Plotly

```js
import Plotly from "plotly.js-dist-min";

const trace = {
  x: data.map((p) => new Date(p.measurement_start)),
  y: [0, 30, 60, 90], // distance bins
  z: zTransposed,       // [bins][profiles]
  type: "heatmap",
  colorscale: "Viridis",
};

const layout = {
  title: "My Chart",
  xaxis: { title: "Time" },
  yaxis: { title: "Distance (m)" },
};

Plotly.newPlot(document.getElementById("chart"), [trace], layout);
```

### Обработка ошибок в API

Все ошибки от `request()` — экземпляры `Error` с текстом из `response.error`.
Используйте паттерн:

```js
try {
  const result = await apiCall();
  // success
} catch (err) {
  errorEl.textContent = err.message;
  errorEl.hidden = false;
}
```

---

## Веб-сервер для разработки

```bash
cd frontend
npm run dev
# → http://localhost:5173
```

API-запросы проксируются через nginx (`https://localhost`) — все сервисы
должны быть запущены в Docker:

```bash
docker compose up -d
```

## Сборка для продакшена

```bash
cd frontend && npm run build
```

Собранные файлы — в `frontend/dist/`. В Docker статика подключается через
volume в nginx: `./frontend/dist:/usr/share/nginx/html`.

## Зависимости npm

| Пакет | Назначение |
|---|---|
| `vite` | Сборщик и dev-сервер |
| `plotly.js-dist-min` | Интерактивные графики |

Не добавляйте лишних зависимостей без необходимости. Vanilla JS + Vite достаточно.
