import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { AppViewState } from "../app-view-state.ts";

export function renderRbacLogin(state: AppViewState) {
  const errorMsg = (state as any).rbacLoginError ?? null;
  const loading = (state as any).rbacLoginLoading ?? false;
  const basePath = state.basePath;

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

  return html`
    <div class="login-container">
      <div class="login-card">
        <div class="login-logo" style="display: flex; align-items: center; justify-content: center; gap: 10px; margin-bottom: 16px;">
          <img src=${basePath ? `${basePath}/logo_h.png` : "/logo_h.png"} alt="ApexOps" style="height: 36px; width: auto; flex-shrink: 0;" />
          <span style="font-weight: 800; font-size: 24px; letter-spacing: 0.5px; color: var(--text-primary);">ApexOps</span>
        </div>
        <div class="login-header">
          <h2 class="login-title">ApexOps 运维平台</h2>
          <p class="login-subtitle">请使用您的账号登录</p>
        </div>

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
