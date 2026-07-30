const API_BASE = "";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getToken() {
  return localStorage.getItem("token");
}

async function request(method, path, options = {}) {
  const { body, formData, params } = options;

  let url = `${API_BASE}${path}`;
  if (params) {
    const qs = new URLSearchParams(
      Object.fromEntries(Object.entries(params).filter(([, v]) => v != null && v !== ""))
    );
    const qstr = qs.toString();
    if (qstr) url += `?${qstr}`;
  }

  const headers = {};
  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  if (formData) {
    // multipart — no Content-Type, let browser set it
  } else if (body != null) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(url, {
    method,
    headers,
    body: formData || (body != null ? JSON.stringify(body) : undefined),
  });

  // Handle 401 — redirect to login
  if (res.status === 401) {
    localStorage.removeItem("token");
    window.location.hash = "#/login";
    throw new Error("Unauthorized");
  }

  // Try to parse JSON; if fails, return text
  const text = await res.text();
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    data = text;
  }

  if (!res.ok) {
    const msg = typeof data === "object" && data.error ? data.error : text;
    throw new Error(msg);
  }

  return data;
}

// ---------------------------------------------------------------------------
// Identity API
// ---------------------------------------------------------------------------

export function register(email, password) {
  return request("POST", "/register", { body: { email, password } });
}

export function login(email, password) {
  return request("POST", "/login", { body: { email, password } });
}

// ---------------------------------------------------------------------------
// Lidar API
// ---------------------------------------------------------------------------

export function listExperiments({ startTime, endTime, limit, offset } = {}) {
  return request("GET", "/api/v1/experiments/list", {
    params: {
      start_time: startTime,
      end_time: endTime,
      limit,
      offset,
    },
  });
}

export function createExperiment(formData) {
  return request("POST", "/api/v1/experiments/create", { formData });
}

export function createTask(body) {
  return request("POST", "/api/v1/experiments/task", { body });
}

export function getTaskStatus(taskID) {
  return request("GET", `/api/v1/tasks/${taskID}`);
}

export function deleteTask(taskID) {
  return request("DELETE", `/api/v1/tasks/${taskID}`);
}

// ---------------------------------------------------------------------------
// Prepared Profiles API
// ---------------------------------------------------------------------------

export function listPreparedExperiments() {
  return request("GET", "/api/v1/prepared-profiles/experiments");
}

export function listPreparedFilters({ experimentId, wavelength, polarization } = {}) {
  return request("GET", "/api/v1/prepared-profiles/filters", {
    params: {
      experiment_id: experimentId,
      wavelength,
      polarization,
    },
  });
}

export function getPreparedProfiles({ experimentId, wavelength, polarization, deviceId } = {}) {
  return request("GET", "/api/v1/prepared-profiles", {
    params: {
      experiment_id: experimentId,
      wavelength,
      polarization,
      device_id: deviceId,
    },
  });
}
