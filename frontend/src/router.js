// Simple hash-based SPA router.

const routes = {};
let currentCleanup = null;
let currentPath = null;

export function route(path, renderFn) {
  routes[path] = renderFn;
}

function navigate(path) {
  window.location.hash = `#${path}`;
}

function getHashPath() {
  const hash = window.location.hash;
  return hash ? hash.slice(1) : "/login";
}

function matchRoute(path) {
  // Exact match first
  if (routes[path]) return { fn: routes[path], params: {} };

  // Dynamic :param routes
  for (const [pattern, fn] of Object.entries(routes)) {
    const parts = pattern.split("/");
    const pathParts = path.split("/");
    if (parts.length !== pathParts.length) continue;

    const params = {};
    let match = true;
    for (let i = 0; i < parts.length; i++) {
      if (parts[i].startsWith(":")) {
        params[parts[i].slice(1)] = pathParts[i];
      } else if (parts[i] !== pathParts[i]) {
        match = false;
        break;
      }
    }
    if (match) return { fn, params };
  }

  return null;
}

function render() {
  const path = getHashPath();
  currentPath = path;

  // Show/hide nav based on auth
  const nav = document.getElementById("nav");
  const token = localStorage.getItem("token");
  nav.hidden = !token;

  if (currentCleanup) {
    currentCleanup();
    currentCleanup = null;
  }

  const match = matchRoute(path);
  if (!match) {
    document.getElementById("content").innerHTML =
      '<div class="error-page"><h2>404</h2><p>Страница не найдена</p><a href="#/experiments">На главную</a></div>';
    return;
  }

  const { fn, params } = match;
  const cleanup = fn(document.getElementById("content"), params);
  if (typeof cleanup === "function") {
    currentCleanup = cleanup;
  }
}

export function initRouter() {
  window.addEventListener("hashchange", render);
  render();
}

import { logout } from "./api.js";

// Logout handler
document.addEventListener("click", (e) => {
  const link = e.target.closest("[data-action='logout']");
  if (link) {
    e.preventDefault();
    // Best-effort server-side revoke of the refresh token.
    const refreshToken = localStorage.getItem("refresh_token");
    if (refreshToken) {
      logout(refreshToken);
    }
    localStorage.removeItem("token");
    localStorage.removeItem("refresh_token");
    navigate("/login");
  }
});
