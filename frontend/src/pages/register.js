import { register } from "../api.js";

export function renderRegister(container) {
  container.innerHTML = `
    <div class="auth-page">
      <div class="card auth-card">
        <h1>Регистрация</h1>
        <form id="register-form">
          <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" required autocomplete="email" />
          </div>
          <div class="form-group">
            <label for="password">Пароль</label>
            <input type="password" id="password" name="password" required minlength="8" autocomplete="new-password" />
            <div class="text-muted mt-8">Минимум 8 символов</div>
          </div>
          <div class="form-group">
            <label for="confirm-password">Подтверждение пароля</label>
            <input type="password" id="confirm-password" name="confirm-password" required autocomplete="new-password" />
          </div>
          <div id="register-error" class="flash flash-error" hidden></div>
          <div id="register-success" class="flash flash-success" hidden></div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary" id="register-btn">Зарегистрироваться</button>
          </div>
        </form>
        <div class="form-footer">
          Уже есть аккаунт? <a href="#/login">Войти</a>
        </div>
      </div>
    </div>
  `;

  const form = document.getElementById("register-form");
  const errorEl = document.getElementById("register-error");
  const successEl = document.getElementById("register-success");
  const btn = document.getElementById("register-btn");

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    errorEl.hidden = true;
    successEl.hidden = true;

    const email = form.email.value.trim();
    const password = form.password.value;
    const confirm = form["confirm-password"].value;

    if (password !== confirm) {
      errorEl.textContent = "Пароли не совпадают";
      errorEl.hidden = false;
      return;
    }

    btn.disabled = true;
    btn.textContent = "Регистрация...";

    try {
      await register(email, password);
      successEl.textContent = "Регистрация успешна! Проверьте email для подтверждения.";
      successEl.hidden = false;
      form.reset();
    } catch (err) {
      errorEl.textContent = err.message;
      errorEl.hidden = false;
    } finally {
      btn.disabled = false;
      btn.textContent = "Зарегистрироваться";
    }
  });
}
