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
          <svg viewBox="0 0 100 100" style="height: 36px; width: 36px; flex-shrink: 0;">
            <!-- Top Left Blue curves -->
            <path d="M 50,5 C 22,5 5,22 5,50 C 5,60 10,70 18,78 L 28,68 C 22,62 18,55 18,50 C 18,32 32,18 50,18 Z" fill="#0070c0"/>
            <path d="M 50,18 C 32,18 18,32 18,50 C 18,55 20,60 25,65 L 35,55 C 32,52 30,50 30,48 C 30,37 37,30 48,30 Z" fill="#0070c0"/>
            <!-- Middle Green curves -->
            <path d="M 50,30 C 37,30 30,37 30,48 C 30,50 32,52 35,55 L 45,45 C 42,42 42,40 42,38 C 42,35 45,32 48,32 Z" fill="#84be3d"/>
            <!-- Bottom Right Blue curves -->
            <path d="M 50,95 C 78,95 95,78 95,50 C 95,40 90,30 82,22 L 72,32 C 78,38 82,45 82,50 C 82,68 68,82 50,82 Z" fill="#0070c0"/>
            <path d="M 50,82 C 68,82 82,68 82,50 C 82,45 80,40 75,35 L 65,45 C 68,48 70,50 70,52 C 70,63 63,70 52,70 Z" fill="#0070c0"/>
            <!-- Middle Green curves Bottom -->
            <path d="M 50,70 C 63,70 70,63 70,52 C 70,50 68,48 65,45 L 55,55 C 58,58 58,60 58,62 C 58,65 55,68 52,68 Z" fill="#84be3d"/>
          </svg>
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
