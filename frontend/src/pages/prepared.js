import { listPreparedExperiments, listPreparedFilters, getPreparedProfiles } from "../api.js";
import Plotly from "plotly.js-dist-min";

export function renderPrepared(container) {
  container.innerHTML = `
    <div class="card">
      <h1>Подготовленные профили</h1>

      <div class="form-row">
        <div class="form-group">
          <label for="prep-experiment">Эксперимент</label>
          <select id="prep-experiment" required>
            <option value="">— загрузка —</option>
          </select>
        </div>
        <div class="form-group">
          <label for="prep-wavelength">Длина волны</label>
          <select id="prep-wavelength">
            <option value="">— все —</option>
          </select>
        </div>
        <div class="form-group">
          <label for="prep-polarization">Поляризация</label>
          <select id="prep-polarization">
            <option value="">— все —</option>
          </select>
        </div>
        <div class="form-group">
          <label for="prep-device">Тип канала</label>
          <select id="prep-device">
            <option value="">— все —</option>
          </select>
        </div>
        <div class="form-group">
          <label for="prep-chart">Тип графика</label>
          <select id="prep-chart">
            <option value="heatmap">Heatmap</option>
            <option value="profile">Profile</option>
            <option value="profile_avg">Profile Avg</option>
          </select>
        </div>
        <div class="form-group">
          <label for="prep-transform">Преобразование</label>
          <select id="prep-transform">
            <option value="raw">Raw</option>
            <option value="pr2">P × r²</option>
            <option value="log10_pr2">log₁₀(P × r²)</option>
            <option value="log10_raw">log₁₀(P)</option>
          </select>
        </div>
      </div>

      <div class="form-actions">
        <button class="btn btn-primary" id="prep-show-btn">Показать</button>
      </div>

      <div id="prep-error" class="flash flash-error" hidden></div>
      <div id="prep-loading" class="loading" hidden>Загрузка...</div>
    </div>

    <div class="card">
      <div id="prep-chart-container"></div>
    </div>
  `;

  const expSelect = document.getElementById("prep-experiment");
  const wlSelect = document.getElementById("prep-wavelength");
  const polSelect = document.getElementById("prep-polarization");
  const devSelect = document.getElementById("prep-device");
  const chartSelect = document.getElementById("prep-chart");
  const transformSelect = document.getElementById("prep-transform");
  const showBtn = document.getElementById("prep-show-btn");
  const errorEl = document.getElementById("prep-error");
  const loadingEl = document.getElementById("prep-loading");
  const chartContainer = document.getElementById("prep-chart-container");

  let currentFilters = null;

  // ---- Load experiments ----
  async function loadExperiments() {
    try {
      const data = await listPreparedExperiments();
      expSelect.innerHTML =
        `<option value="">— выберите —</option>` +
        data.map((e) => `<option value="${e.experiment_id}">${escapeHtml(e.title)} (${formatDate(e.experiment_start)} — ${formatDate(e.experiment_end)})</option>`).join("");
    } catch (err) {
      expSelect.innerHTML = `<option value="">Ошибка: ${err.message}</option>`;
    }
  }
  loadExperiments();

  // ---- On experiment change ----
  expSelect.addEventListener("change", async () => {
    const expId = expSelect.value;
    wlSelect.innerHTML = `<option value="">— все —</option>`;
    polSelect.innerHTML = `<option value="">— все —</option>`;
    devSelect.innerHTML = `<option value="">— все —</option>`;
    chartContainer.innerHTML = "";
    if (!expId) return;

    try {
      const filters = await listPreparedFilters({ experimentId: expId });
      currentFilters = filters;
      wlSelect.innerHTML =
        `<option value="">— все —</option>` +
        filters.wavelengths.map((w) => `<option value="${w}">${w}</option>`).join("");
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    }
  });

  // ---- On wavelength change ----
  wlSelect.addEventListener("change", () => {
    updatePolarizations();
  });

  function updatePolarizations() {
    const wl = wlSelect.value;
    polSelect.innerHTML = `<option value="">— все —</option>`;
    devSelect.innerHTML = `<option value="">— все —</option>`;

    if (!currentFilters) return;

    let pols = currentFilters.polarizations;
    polSelect.innerHTML =
      `<option value="">— все —</option>` +
      pols.map((p) => `<option value="${p}">${p}</option>`).join("");
  }

  // ---- On polarization change ----
  polSelect.addEventListener("change", () => {
    updateDeviceIDs();
  });

  async function updateDeviceIDs() {
    const expId = expSelect.value;
    const wl = wlSelect.value;
    const pol = polSelect.value;
    devSelect.innerHTML = `<option value="">— все —</option>`;

    if (!expId) return;

    try {
      const filters = await listPreparedFilters({
        experimentId: expId,
        wavelength: wl || undefined,
        polarization: pol || undefined,
      });
      devSelect.innerHTML =
        `<option value="">— все —</option>` +
        (filters.device_ids || []).map((d) => `<option value="${d}">${d}</option>`).join("");
    } catch {
      // ignore
    }
  }

  // ---- Show chart ----
  showBtn.addEventListener("click", async () => {
    const expId = expSelect.value;
    if (!expId) {
      errorEl.textContent = "Выберите эксперимент";
      errorEl.hidden = false;
      return;
    }

    const wl = wlSelect.value;
    const pol = polSelect.value;
    const dev = devSelect.value;
    const chartType = chartSelect.value;
    const transform = transformSelect.value;

    errorEl.hidden = true;
    loadingEl.hidden = false;
    chartContainer.innerHTML = "";

    try {
      const data = await getPreparedProfiles({
        experimentId: expId,
        wavelength: wl || undefined,
        polarization: pol || undefined,
        deviceId: dev || undefined,
      });

      const profiles = data.profiles || [];
      if (profiles.length === 0) {
        errorEl.textContent = "Нет данных для выбранных фильтров";
        errorEl.hidden = false;
        return;
      }

      renderChart(profiles, chartType, transform, chartContainer);
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    } finally {
      loadingEl.hidden = true;
    }
  });

  // -----------------------------------------------------------------------
  // Transform helpers
  // -----------------------------------------------------------------------

  function applyTransform(value, binIdx, binWidth, transform) {
    const r = binIdx * binWidth/1e3;
    switch (transform) {
      case "pr2":
        return value * r * r;
      case "log10_pr2":
        if (r <= 0 || value <= 0) return null;
        return Math.asinh(value * r * r/1e-6);//Math.log10(value * r * r);
      case "log10_raw":
        if (value <= 0) return null;
        return Math.asinh(value/1e-6);//Math.log10(value);
      default:
        return value;
    }
  }

  function transformArray(data, binWidth, transform) {
    if (transform === "raw") return data;
    return data.map((v, i) => applyTransform(v, i, binWidth, transform));
  }

  // -----------------------------------------------------------------------
  // Chart rendering
  // -----------------------------------------------------------------------

  function renderChart(profiles, chartType, transform, container) {
    const binWidth = profiles[0].bin_width || 30;
    const maxBins = Math.max(...profiles.map((p) => (p.data || []).length));
    const yAxis = Array.from({ length: maxBins }, (_, i) => i * binWidth);

    // Pre-transform all profiles
    const transformed = profiles.map((p) => {
      const d = p.data || [];
      return { ...p, data: transformArray(d, binWidth, transform) };
    });

    const transformLabel = getTransformLabel(transform);

    // Sort profiles by measurement time for a meaningful timeline
    transformed.sort((a, b) => {
      const ta = new Date(a.measurement_start).getTime();
      const tb = new Date(b.measurement_start).getTime();
      if (isNaN(ta) || isNaN(tb)) return 0;
      return ta - tb;
    });

    if (chartType === "heatmap") {
      // Build matrix: rows = distance bins, cols = profiles
      // This way Plotly maps: z[binIdx][profileIdx] → x[profileIdx], y[binIdx]
      const zTransposed = [];
      for (let b = 0; b < maxBins; b++) {
        const row = [];
        for (let p = 0; p < transformed.length; p++) {
          const d = transformed[p].data || [];
          row.push(b < d.length ? d[b] : null);
        }
        zTransposed.push(row);
      }

      const trace = {
        z: zTransposed,
        x: transformed.map((p) => new Date(p.measurement_start)),
        y: yAxis,
        type: "heatmap",
        colorscale: "Viridis",
        colorbar: { title: transformLabel },
      };

      const layout = {
        title: `Heatmap (${transformed.length} profiles) — ${transformLabel}`,
        xaxis: { title: "Time" },
        yaxis: { title: "Distance (m)" },
        margin: { l: 60, r: 40, t: 50, b: 60 },
      };

      Plotly.newPlot(container, [trace], layout);
    } else if (chartType === "profile") {
      const traces = transformed.map((p, i) => ({
        x: p.data || [],
        y: yAxis.slice(0, (p.data || []).length),
        type: "scatter",
        mode: "lines",
        name: `${p.wavelength}nm ${p.polarization} ${p.device_id} #${i}`,
        line: { width: 1 },
      }));

      const layout = {
        title: `Profiles (${transformed.length}) — ${transformLabel}`,
        xaxis: { title: transformLabel },
        yaxis: { title: "Distance (m)" },
        margin: { l: 60, r: 40, t: 50, b: 60 },
      };

      Plotly.newPlot(container, traces, layout);
    } else if (chartType === "profile_avg") {
      const maxLen = Math.max(...transformed.map((p) => (p.data || []).length));
      const sums = Array(maxLen).fill(0);
      const counts = Array(maxLen).fill(0);
      transformed.forEach((p) => {
        const d = p.data || [];
        for (let i = 0; i < d.length; i++) {
          if (d[i] !== null) {
            sums[i] += d[i];
            counts[i]++;
          }
        }
      });
      const avg = sums.map((s, i) => (counts[i] > 0 ? s / counts[i] : null));

      const trace = {
        x: avg,
        y: yAxis.slice(0, avg.length),
        type: "scatter",
        mode: "lines",
        name: "Average",
        line: { width: 2, color: "red" },
      };

      const layout = {
        title: `Average Profile (${transformed.length} profiles) — ${transformLabel}`,
        xaxis: { title: `Mean ${transformLabel}` },
        yaxis: { title: "Distance (m)" },
        margin: { l: 60, r: 40, t: 50, b: 60 },
      };

      Plotly.newPlot(container, [trace], layout);
    }
  }

  function getTransformLabel(transform) {
    const labels = {
      raw: "Signal",
      pr2: "P × r²",
      log10_pr2: "log₁₀(P × r²)",
      log10_raw: "log₁₀(P)",
    };
    return labels[transform] || "Signal";
  }
}


// ---- Helpers ----

function escapeHtml(str) {
  if (!str) return "";
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
