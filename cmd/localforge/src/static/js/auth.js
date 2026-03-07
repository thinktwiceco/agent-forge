function desiredNextPath() {
  const current = window.location.pathname + window.location.search;
  if (window.location.pathname !== "/login") {
    return current;
  }

  const params = new URLSearchParams(window.location.search);
  return params.get("next") || "/";
}

async function getAuthStatus() {
  try {
    const next = encodeURIComponent(desiredNextPath());
    const res = await fetch(`/api/auth/me?next=${next}`);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

async function logout() {
  await fetch("/api/auth/logout", { method: "POST" });
  window.location.href = "/login";
}

export async function setupAuthUI() {
  const status = await getAuthStatus();
  const buttons = document.querySelectorAll("[data-auth-logout]");

  if (!status?.enabled) {
    buttons.forEach((button) => {
      button.hidden = true;
    });
    return status;
  }

  buttons.forEach((button) => {
    button.hidden = false;
    if (button.dataset.authBound === "true") return;
    button.dataset.authBound = "true";
    button.addEventListener("click", async () => {
      button.disabled = true;
      try {
        await logout();
      } finally {
        button.disabled = false;
      }
    });
  });

  return status;
}

export async function setupLoginForm() {
  const form = document.getElementById("login-form");
  const message = document.getElementById("login-message");
  if (!form || !message) return;

  const status = await getAuthStatus();
  if (status?.enabled === false) {
    window.location.replace("/");
    return;
  }
  if (status?.authenticated) {
    window.location.replace(status.next || "/");
    return;
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    message.textContent = "";

    const username = document.getElementById("login-username")?.value || "";
    const password = document.getElementById("login-password")?.value || "";
    const submit = form.querySelector('button[type="submit"]');
    if (submit) submit.disabled = true;

    try {
      const next = encodeURIComponent(desiredNextPath());
      const res = await fetch(`/api/auth/login?next=${next}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      const payload = await res.json().catch(() => ({}));
      if (!res.ok) {
        message.textContent = payload.error || "Login failed";
        return;
      }

      window.location.replace(payload.next || "/");
    } catch {
      message.textContent = "Network error";
    } finally {
      if (submit) submit.disabled = false;
    }
  });
}
