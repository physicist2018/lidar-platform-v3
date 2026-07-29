import { listExperiments } from "../api.js";
import { getState, setState } from "../store.js";

export function renderExperiments(container) {
  container.innerHTML = `
    <div class="card">
      <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;">
        <h1 style="margin:0;">Эксперименты</h1>
        <a href="#/upload" class="btn btn-primary">Загрузить</a>
      </div>

      <form id="experiments-filter" style="display:flex;gap:12px;align-items:end;flex-wrap:wrap;margin-bottom:16px;padding:12px;background:var(--color-bg);border-radius:var(--radius);">
        <div class="form-group" style="margin:0;flex:1;min-width:160px;">
          <label for="start-time">От</label>
          <input type="datetime-local" id="start-time" />
        </div>
        <div class="form-group" style="margin:0;flex:1;min-width:160px;">
          <label for="end-time">До</label>
          <input type="datetime-local" id="end-time" />
        </div>
        <button type="submit" class="btn btn-outline" style="margin-bottom:0;">Фильтр</button>
      </form>

      <div id="experiments-loading" class="loading">Загрузка...</div>
      <div id="experiments-list" class="table-wrap" hidden>
        <table>
          <thead>
            <tr>
              <th>Название</th>
              <th>Угол зенита</th>
              <th>Координаты</th>
              <th>Создан</th>
            </tr>
          </thead>
          <tbody id="experiments-tbody"></tbody>
        </table>
      </div>
      <div id="experiments-empty" class="empty-state" hidden>
        <p>Экспериментов пока нет.</p>
        <a href="#/upload" class="btn btn-primary mt-16">Загрузить первый эксперимент</a>
      </div>
    </div>
  `;

  const loadingEl = document.getElementById("experiments-loading");
  const listEl = document.getElementById("experiments-list");
  const emptyEl = document.getElementById("experiments-empty");
  const tbody = document.getElementById("experiments-tbody");
  const filterForm = document.getElementById("experiments-filter");

  async function load(params = {}) {
    loadingEl.hidden = false;
    listEl.hidden = true;
    emptyEl.hidden = true;

    try {
      const data = await listExperiments(params);
      const experiments = data.experiments || [];
      setState({ experiments });

      if (experiments.length === 0) {
        emptyEl.hidden = false;
      } else {
        tbody.innerHTML = experiments
          .map(
            (exp) => `
              <tr data-id="${exp.id}">
                <td><strong>${escapeHtml(exp.title)}</strong></td>
                <td>${exp.zenith_angle}°</td>
                <td>${exp.latitude}, ${exp.longitude}</td>
                <td class="text-muted">${formatDate(exp.created_at)}</td>
              </tr>
            `
          )
          .join("");

        // Click on row → detail page
        tbody.querySelectorAll("tr").forEach((tr) => {
          tr.addEventListener("click", () => {
            window.location.hash = `#/experiments/${tr.dataset.id}`;
          });
        });

        listEl.hidden = false;
      }
    } catch (err) {
      tbody.innerHTML = `<tr><td colspan="4" style="color:var(--color-danger);">Ошибка: ${err.message}</td></tr>`;
      listEl.hidden = false;
    } finally {
      loadingEl.hidden = true;
    }
  }

  filterForm.addEventListener("submit", (e) => {
    e.preventDefault();
    const startVal = document.getElementById("start-time").value;
    const endVal = document.getElementById("end-time").value;
    load({
      startTime: startVal ? new Date(startVal).toISOString() : undefined,
      endTime: endVal ? new Date(endVal).toISOString() : undefined,
    });
  });

  load();
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
