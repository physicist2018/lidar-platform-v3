export function renderVerified(container) {
  const params = new URLSearchParams(window.location.search);
  const status = params.get("status");
  const reason = params.get("reason");

  let title, message, extra;

  if (status === "ok") {
    title = "Email подтверждён";
    message = "Ваш email успешно подтверждён. Теперь вы можете войти в систему.";
    extra = '<a href="#/login" class="btn btn-primary">Войти</a>';
  } else if (status === "error" && reason === "already_verified") {
    title = "Уже подтверждён";
    message = "Этот email уже был подтверждён ранее.";
    extra = '<a href="#/login" class="btn btn-primary">Войти</a>';
  } else if (status === "error" && reason === "invalid_token") {
    title = "Ошибка верификации";
    message = "Ссылка для подтверждения недействительна или истекла. Попробуйте зарегистрироваться заново.";
    extra = '<a href="#/register" class="btn btn-primary">Зарегистрироваться</a>';
  } else {
    title = "Ошибка";
    message = "Что-то пошло не так. Попробуйте позже.";
    extra = '<a href="#/login" class="btn btn-outline">На главную</a>';
  }

  container.innerHTML = `
    <div class="auth-page">
      <div class="card auth-card" style="text-align:center;">
        <h1>${title}</h1>
        <p style="margin-bottom:24px;">${message}</p>
        ${extra}
      </div>
    </div>
  `;
}
