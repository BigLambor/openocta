import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { AppViewState } from "../app-view-state.ts";

export function renderRbacLogin(state: AppViewState) {
  const errorMsg = (state as any).rbacLoginError ?? null;
  const loading = (state as any).rbacLoginLoading ?? false;
  const needsSetup = (state as any).rbacNeedsSetup === true;
  const basePath = state.basePath;

  const handleSetupSubmit = (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const usernameInput = form.querySelector("#setup-username") as HTMLInputElement;
    const passwordInput = form.querySelector("#setup-password") as HTMLInputElement;
    const confirmInput = form.querySelector("#setup-password-confirm") as HTMLInputElement;
    const username = usernameInput.value.trim();
    const password = passwordInput.value;
    const confirm = confirmInput.value;

    if (!username || !password) {
      return;
    }
    if (password !== confirm) {
      (state as any).rbacLoginError = "两次输入的密码不一致";
      return;
    }
    if (password.length < 8) {
      (state as any).rbacLoginError = "密码长度至少 8 位";
      return;
    }

    if ((state as any).handleRbacSetup) {
      void (state as any).handleRbacSetup(username, password);
    }
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const usernameInput = form.querySelector("#login-username") as HTMLInputElement;
    const passwordInput = form.querySelector("#login-password") as HTMLInputElement;
    const username = usernameInput.value.trim();
    const password = passwordInput.value.trim();

    if (!username || !password) {
      return;
    }

    if ((state as any).handleRbacLogin) {
      void (state as any).handleRbacLogin(username, password);
    }
  };

  if (needsSetup) {
    return html`
      <div class="login-container">
        <div class="login-card">
          <div class="login-logo">
            <img src=${basePath ? `${basePath}/logo_h.png` : "/logo_h.png"} alt="ApexOps" />
            <span class="login-logo__name">ApexOps</span>
          </div>
          <p class="login-subtitle">首次启动，请创建管理员账号</p>

          ${errorMsg
            ? html`
                <div class="login-error" role="alert">
                  <span>${icons.x}</span>
                  <span>${errorMsg}</span>
                </div>
              `
            : nothing}

          <form class="login-form" @submit=${handleSetupSubmit}>
            <div>
              <label class="form-label" for="setup-username">管理员用户名</label>
              <input
                type="text"
                id="setup-username"
                class="login-input"
                required
                placeholder="admin"
                value="admin"
                ?disabled=${loading}
              />
            </div>
            <div>
              <label class="form-label" for="setup-password">密码</label>
              <input
                type="password"
                id="setup-password"
                class="login-input"
                required
                minlength="8"
                placeholder="至少 8 位"
                ?disabled=${loading}
              />
            </div>
            <div>
              <label class="form-label" for="setup-password-confirm">确认密码</label>
              <input
                type="password"
                id="setup-password-confirm"
                class="login-input"
                required
                minlength="8"
                placeholder="再次输入密码"
                ?disabled=${loading}
              />
            </div>
            <button type="submit" class="login-button" ?disabled=${loading}>
              ${loading ? "正在初始化…" : "创建管理员并登录"}
            </button>
          </form>
        </div>
      </div>
    `;
  }

  return html`
    <div class="login-container">
      <div class="login-card">
        <div class="login-logo">
          <img src=${basePath ? `${basePath}/logo_h.png` : "/logo_h.png"} alt="ApexOps" />
          <span class="login-logo__name">ApexOps</span>
        </div>
        <p class="login-subtitle">请使用您的账号登录</p>

        ${errorMsg
          ? html`
              <div class="login-error" role="alert">
                <span>${icons.x}</span>
                <span>${errorMsg}</span>
              </div>
            `
          : nothing}

        <form class="login-form" @submit=${handleSubmit}>
          <div>
            <label class="form-label" for="login-username">用户名</label>
            <input
              type="text"
              id="login-username"
              class="login-input"
              required
              placeholder="用户名"
              ?disabled=${loading}
            />
          </div>
          <div>
            <label class="form-label" for="login-password">密码</label>
            <input
              type="password"
              id="login-password"
              class="login-input"
              required
              placeholder="密码"
              ?disabled=${loading}
            />
          </div>
          <button type="submit" class="login-button" ?disabled=${loading}>
            ${loading ? "正在登录…" : "登录"}
          </button>
        </form>
      </div>
    </div>
  `;
}
