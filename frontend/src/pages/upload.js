import { createExperiment } from "../api.js";

export function renderUpload(container) {
  container.innerHTML = `
    <div>
      <a href="#/experiments" class="btn btn-outline mb-16">← К списку</a>

      <div class="card">
        <h1>Загрузить эксперимент</h1>
        <form id="upload-form">
          <div class="form-group">
            <label for="title">Название *</label>
            <input type="text" id="title" name="title" required />
          </div>

          <div class="form-group">
            <label for="comments">Комментарий</label>
            <textarea id="comments" name="comments"></textarea>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="zenith">Угол зенита (°) *</label>
              <input type="number" id="zenith" name="zenith_angle" step="0.1" required />
            </div>
            <div class="form-group">
              <label for="lat">Широта *</label>
              <input type="number" id="lat" name="latitude" step="0.01" required />
            </div>
            <div class="form-group">
              <label for="lng">Долгота *</label>
              <input type="number" id="lng" name="longitude" step="0.01" required />
            </div>
          </div>

          <div class="form-group">
            <label for="exp-files">Архив с данными (ZIP) *</label>
            <input type="file" id="exp-files" name="experiment_files" accept=".zip" required />
          </div>

          <div class="form-group">
            <label for="bg-file">Фоновый файл (LICEL, опционально)</label>
            <input type="file" id="bg-file" name="background" accept="*.*" />
          </div>

          <div class="form-group">
            <label for="meteo-file">Метеофайл (CSV, опционально)</label>
            <input type="file" id="meteo-file" name="meteo" accept=".csv,.txt" />
          </div>

          <div id="upload-error" class="flash flash-error" hidden></div>
          <div id="upload-success" class="flash flash-success" hidden></div>

          <div class="form-actions">
            <button type="submit" class="btn btn-primary" id="upload-btn">Загрузить</button>
          </div>
        </form>
      </div>
    </div>
  `;

  const form = document.getElementById("upload-form");
  const errorEl = document.getElementById("upload-error");
  const successEl = document.getElementById("upload-success");
  const btn = document.getElementById("upload-btn");

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    errorEl.hidden = true;
    successEl.hidden = true;

    const fd = new FormData(form);

    // Validate required numeric fields
    const zenith = parseFloat(fd.get("zenith_angle"));
    const lat = parseFloat(fd.get("latitude"));
    const lng = parseFloat(fd.get("longitude"));

    if (isNaN(zenith) || isNaN(lat) || isNaN(lng)) {
      errorEl.textContent = "Заполните все обязательные поля";
      errorEl.hidden = false;
      return;
    }

    // Validate file
    const files = fd.get("experiment_files");
    if (!files || files.size === 0) {
      errorEl.textContent = "Выберите архив с данными";
      errorEl.hidden = false;
      return;
    }

    btn.disabled = true;
    btn.textContent = "Загрузка...";

    try {
      const result = await createExperiment(fd);
      successEl.textContent = `Эксперимент "${result.title}" успешно загружен!`;
      successEl.hidden = false;
      form.reset();

      // Redirect to experiment detail after a short delay
      setTimeout(() => {
        window.location.hash = `#/experiments/${result.id}`;
      }, 1500);
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    } finally {
      btn.disabled = false;
      btn.textContent = "Загрузить";
    }
  });
}
