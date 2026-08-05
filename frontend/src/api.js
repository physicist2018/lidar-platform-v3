const API_BASE = "";

const TOKEN_KEY = "token";
const REFRESH_TOKEN_KEY = "refresh_token";

// ---------------------------------------------------------------------------
// Token helpers
// ---------------------------------------------------------------------------

function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// ---------------------------------------------------------------------------
// Silent refresh (single-flight)
// ---------------------------------------------------------------------------

// Only one POST /refresh at a time; concurrent 401-triggered refreshes
// await the same promise.
let refreshPromise = null;

async function refreshTokens() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) throw new Error("No refresh token");

  const res = await fetch(`${API_BASE}/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!res.ok) throw new Error("Refresh failed");

  const data = await res.json();
  localStorage.setItem(TOKEN_KEY, data.token);
  localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token);
  return data;
}

// Endpoints that must never trigger a refresh on 401.
function isAuthPath(path) {
  return path === "/login" || path === "/refresh" || path === "/logout";
}

// ---------------------------------------------------------------------------
// Request core
// ---------------------------------------------------------------------------

async function request(method, path, options = {}) {
  const { body, formData, params, retried } = options;

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

  // Access token expired → refresh once and retry the request.
  // Only for authenticated API calls; auth endpoints are excluded.
  if (res.status === 401 && token && !retried && !isAuthPath(path)) {
    try {
      if (!refreshPromise) {
        refreshPromise = refreshTokens().finally(() => {
          refreshPromise = null;
        });
      }
      await refreshPromise;
      return request(method, path, { ...options, retried: true });
    } catch {
      clearTokens();
      window.location.hash = "#/login";
      throw new Error("Unauthorized");
    }
  }

  // Handle 401 for auth endpoints (bad credentials, invalid refresh token, ...)
  if (res.status === 401) {
    clearTokens();
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

// Best-effort server-side revoke; the local session is cleared regardless.
export function logout(refreshToken) {
  return fetch(`${API_BASE}/logout`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  }).catch(() => {});
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
