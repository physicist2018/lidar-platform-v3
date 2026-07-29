import { login } from "../api.js";

export function renderLogin(container) {
  // Already logged in — redirect
  if (localStorage.getItem("token")) {
    window.location.hash = "#/experiments";
    return;
  }

  container.innerHTML = `
    <div class="auth-page">
      <div class="card auth-card">
        <h1>Вход</h1>
        <form id="login-form">
          <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" required autocomplete="email" />
          </div>
          <div class="form-group">
            <label for="password">Пароль</label>
            <input type="password" id="password" name="password" required autocomplete="current-password" />
          </div>
          <div id="login-error" class="flash flash-error" hidden></div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary" id="login-btn">Войти</button>
          </div>
        </form>
        <div class="form-footer">
          Нет аккаунта? <a href="#/register">Зарегистрироваться</a>
        </div>
      </div>
    </div>
  `;

  const form = document.getElementById("login-form");
  const errorEl = document.getElementById("login-error");
  const btn = document.getElementById("login-btn");

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    errorEl.hidden = true;

    const email = form.email.value.trim();
    const password = form.password.value;

    btn.disabled = true;
    btn.textContent = "Вход...";

    try {
      const result = await login(email, password);
      localStorage.setItem("token", result.token);
      window.location.hash = "#/experiments";
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    } finally {
      btn.disabled = false;
      btn.textContent = "Войти";
    }
  });
}
