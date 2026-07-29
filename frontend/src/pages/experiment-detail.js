import { listExperiments, createTask, getTaskStatus } from "../api.js";
import { getState, setState } from "../store.js";

// Module-level interval tracker to avoid accumulating intervals.
let taskRefreshInterval = null;

// Track task IDs per experiment in localStorage
function getTaskIds(expId) {
  try {
    return JSON.parse(localStorage.getItem(`tasks_${expId}`) || "[]");
  } catch {
    return [];
  }
}

function addTaskId(expId, taskId) {
  const ids = getTaskIds(expId);
  if (!ids.includes(taskId)) {
    ids.push(taskId);
    localStorage.setItem(`tasks_${expId}`, JSON.stringify(ids));
  }
}

export async function renderExperimentDetail(container, params) {
  const expId = params.id;

  // Clear any previous interval from a previous experiment detail visit.
  if (taskRefreshInterval) {
    clearInterval(taskRefreshInterval);
    taskRefreshInterval = null;
  }

  container.innerHTML = `<div class="loading">Загрузка...</div>`;

  let exp = getState().experiments.find((e) => e.id === expId);
  if (!exp) {
    try {
      const data = await listExperiments({});
      const found = (data.experiments || []).find((e) => e.id === expId);
      if (!found) {
        container.innerHTML = `<div class="error-page"><h2>404</h2><p>Эксперимент не найден</p><a href="#/experiments">К списку</a></div>`;
        return;
      }
      exp = found;
    } catch (err) {
      container.innerHTML = `<div class="error-page"><p style="color:var(--color-danger)">Ошибка: ${err.message}</p><a href="#/experiments">Назад</a></div>`;
      return;
    }
  }

  setState({ currentExperiment: exp });

  container.innerHTML = `
    <div>
      <a href="#/experiments" class="btn btn-outline mb-16">← К списку</a>

      <div class="card">
        <h1>${escapeHtml(exp.title)}</h1>
        <div class="detail-grid">
          <div class="detail-item">
            <dt>ID</dt>
            <dd style="font-size:0.8rem;word-break:break-all;">${exp.id}</dd>
          </div>
          <div class="detail-item">
            <dt>Угол зенита</dt>
            <dd>${exp.zenith_angle}°</dd>
          </div>
          <div class="detail-item">
            <dt>Координаты</dt>
            <dd>${exp.latitude}, ${exp.longitude}</dd>
          </div>
          <div class="detail-item">
            <dt>Время измерения</dt>
            <dd>${formatDate(exp.experiment_start)} — ${formatDate(exp.experiment_end)}</dd>
          </div>
          <div class="detail-item">
            <dt>Создан</dt>
            <dd>${formatDate(exp.created_at)}</dd>
          </div>
          ${exp.comments ? `<div class="detail-item" style="grid-column:1/-1;">
            <dt>Комментарий</dt>
            <dd>${escapeHtml(exp.comments)}</dd>
          </div>` : ""}
        </div>
      </div>

      <div class="card">
        <h2>Создать задачу</h2>
        <form id="task-form">
          <div class="form-group">
            <label for="task-subject">Тип задачи</label>
            <select id="task-subject" name="subject" required>
              <option value="">— выберите —</option>
              <option value="lidar.task.prepare_experiment">Обработка фона (prepare)</option>
            </select>
          </div>

          <div id="task-params" hidden>
            <div class="form-group">
              <label for="bg-type">Тип вычета фона</label>
              <select id="bg-type" name="background_type">
                <option value="mean">Среднее хвоста (mean)</option>
                <option value="file">Фоновый профиль (file)</option>
              </select>
            </div>
            <div class="form-row">
              <div class="form-group">
                <label for="bg-from">Расстояние для mean (м)</label>
                <input type="number" id="bg-from" name="background_from" value="80000" step="1000" />
              </div>
              <div class="form-group">
                <label for="trim-from">Обрезка (м)</label>
                <input type="number" id="trim-from" name="trim_from" value="20000" step="1000" />
              </div>
            </div>
          </div>

          <div id="task-error" class="flash flash-error" hidden></div>
          <div id="task-success" class="flash flash-success" hidden></div>

          <div class="form-actions">
            <button type="submit" class="btn btn-primary" id="task-btn">Запустить задачу</button>
          </div>
        </form>
      </div>

      <div class="card">
        <h2>Задачи</h2>
        <div id="tasks-list">
          <div class="empty-state">Задач пока нет</div>
        </div>
      </div>
    </div>
  `;

  // --- Task form logic ---
  const taskForm = document.getElementById("task-form");
  const subjectSelect = document.getElementById("task-subject");
  const taskParamsDiv = document.getElementById("task-params");
  const taskError = document.getElementById("task-error");
  const taskSuccess = document.getElementById("task-success");
  const taskBtn = document.getElementById("task-btn");

  subjectSelect.addEventListener("change", () => {
    taskParamsDiv.hidden = subjectSelect.value === "";
  });

  taskForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    taskError.hidden = true;
    taskSuccess.hidden = true;

    const subject = subjectSelect.value;
    if (!subject) return;

    const bgType = document.getElementById("bg-type").value;
    const bgFrom = parseFloat(document.getElementById("bg-from").value) || 0;
    const trimFrom = parseFloat(document.getElementById("trim-from").value) || 0;

    const payload = {
      experiment_id: expId,
      background_type: bgType,
      background_from: bgFrom,
      trim_from: trimFrom,
    };

    taskBtn.disabled = true;
    taskBtn.textContent = "Создание...";

    try {
      const result = await createTask({
        subject,
        task_type: "prepare_experiment",
        payload,
      });
      addTaskId(expId, result.task_id);
      taskSuccess.textContent = `Задача создана! ID: ${result.task_id}`;
      taskSuccess.hidden = false;
      // Refresh task list
      renderTasks(expId);
    } catch (err) {
      taskError.textContent = err.message;
      taskError.hidden = false;
    } finally {
      taskBtn.disabled = false;
      taskBtn.textContent = "Запустить задачу";
    }
  });

  // --- Render tasks ---
  renderTasks(expId);

  // Auto-refresh task statuses every 5 seconds.
  taskRefreshInterval = setInterval(() => renderTasks(expId), 5000);
}

async function renderTasks(expId) {
  const container = document.getElementById("tasks-list");
  if (!container) return;

  const taskIds = getTaskIds(expId);
  if (taskIds.length === 0) {
    container.innerHTML = '<div class="empty-state">Задач пока нет</div>';
    return;
  }

  // Fetch statuses for all task IDs
  const tasks = [];
  for (const id of taskIds) {
    try {
      const status = await getTaskStatus(id);
      tasks.push(status);
    } catch {
      tasks.push({ id, status: "unknown", error_message: "Не удалось получить статус" });
    }
  }

  container.innerHTML = tasks
    .map(
      (t) => `
        <div class="task-item">
          <div>
            <div style="font-size:0.85rem;word-break:break-all;">${t.id}</div>
            <div class="text-muted" style="font-size:0.8rem;">${t.subject || ""}</div>
          </div>
          <div style="display:flex;align-items:center;gap:12px;">
            <span class="badge badge-${t.status}">${statusLabel(t.status)}</span>
            ${t.error_message ? `<span class="text-muted" style="font-size:0.8rem;max-width:200px;">${escapeHtml(t.error_message)}</span>` : ""}
          </div>
        </div>
      `
    )
    .join("");
}

function statusLabel(status) {
  const labels = {
    pending: "Ожидает",
    processing: "В работе",
    completed: "Готово",
    failed: "Ошибка",
  };
  return labels[status] || status;
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function formatDate(iso) {
  if (!iso) return "-";
  const d = new Date(iso);
  return d.toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
