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
    <style>
      .login-container {
        display: flex;
        align-items: center;
        justify-content: center;
        min-height: 100vh;
        width: 100vw;
        background: radial-gradient(circle at 50% 50%, #0d1527 0%, #040814 100%);
        position: relative;
        overflow: hidden;
        font-family: 'Outfit', 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      }

      /* Neon glow effects */
      .login-container::before {
        content: "";
        position: absolute;
        width: 400px;
        height: 400px;
        background: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, transparent 70%);
        top: 15%;
        left: 20%;
        border-radius: 50%;
        filter: blur(40px);
        pointer-events: none;
      }

      .login-container::after {
        content: "";
        position: absolute;
        width: 500px;
        height: 500px;
        background: radial-gradient(circle, rgba(168, 85, 247, 0.1) 0%, transparent 70%);
        bottom: 10%;
        right: 15%;
        border-radius: 50%;
        filter: blur(50px);
        pointer-events: none;
      }

      .login-card {
        background: rgba(10, 17, 38, 0.7);
        backdrop-filter: blur(16px) saturate(180%);
        -webkit-backdrop-filter: blur(16px) saturate(180%);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 16px;
        padding: 40px;
        width: 100%;
        max-width: 420px;
        box-shadow: 0 20px 50px rgba(0, 0, 0, 0.4), 
                    inset 0 1px 0 rgba(255, 255, 255, 0.1);
        z-index: 10;
        animation: fadeIn 0.6s cubic-bezier(0.16, 1, 0.3, 1);
        display: flex;
        flex-direction: column;
        align-items: center;
      }

      @keyframes fadeIn {
        from {
          opacity: 0;
          transform: translateY(20px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }

      .login-logo {
        margin-bottom: 24px;
        display: flex;
        justify-content: center;
      }

      .login-logo img {
        height: 42px;
        object-fit: contain;
      }

      .login-header {
        text-align: center;
        margin-bottom: 32px;
      }

      .login-title {
        font-size: 20px;
        font-weight: 600;
        color: #f3f4f6;
        margin-bottom: 8px;
        letter-spacing: -0.5px;
      }

      .login-subtitle {
        font-size: 13px;
        color: #9ca3af;
      }

      .login-form {
        width: 100%;
        display: flex;
        flex-direction: column;
        gap: 20px;
      }

      .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
      }

      .form-label {
        font-size: 12px;
        font-weight: 500;
        color: #d1d5db;
        text-transform: uppercase;
        letter-spacing: 0.5px;
      }

      .input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
      }

      .input-icon {
        position: absolute;
        left: 12px;
        color: #6b7280;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .input-icon svg {
        width: 16px;
        height: 16px;
      }

      .login-input {
        width: 100%;
        background: rgba(17, 24, 39, 0.6);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        padding: 11px 12px 11px 40px;
        color: #f9fafb;
        font-size: 14px;
        transition: all 0.2s ease;
        outline: none;
      }

      .login-input:focus {
        border-color: #6366f1;
        box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
        background: rgba(17, 24, 39, 0.8);
      }

      .login-input::placeholder {
        color: #4b5563;
      }

      .login-button {
        width: 100%;
        background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
        border: none;
        border-radius: 8px;
        padding: 12px;
        color: #ffffff;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
      }

      .login-button:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
        filter: brightness(1.1);
      }

      .login-button:active:not(:disabled) {
        transform: translateY(1px);
      }

      .login-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .login-error {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        border-radius: 8px;
        padding: 12px;
        width: 100%;
        margin-bottom: 20px;
        display: flex;
        align-items: flex-start;
        gap: 10px;
        animation: shake 0.4s ease;
      }

      @keyframes shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-5px); }
        75% { transform: translateX(5px); }
      }

      .error-icon {
        color: #ef4444;
        margin-top: 1px;
        display: flex;
      }

      .error-text {
        font-size: 13px;
        color: #fca5a5;
        line-height: 1.4;
      }

      .spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-radius: 50%;
        border-top-color: #ffffff;
        animation: spin 0.8s linear infinite;
      }

      @keyframes spin {
        to { transform: rotate(360deg); }
      }
    </style>
    <div class="login-container">
      <div class="login-card">
        <div class="login-logo">
          <img
            src=${basePath ? `${basePath}/logo_h.png` : "/logo_h.png"}
            alt="OpenOcta"
          />
        </div>
        <div class="login-header">
          <h2 class="login-title">智能运维管理平台</h2>
          <p class="login-subtitle">请使用您的 OpenOcta 账号登录</p>
        </div>

        ${errorMsg
          ? html`
              <div class="login-error" role="alert">
                <span class="error-icon">${icons.x}</span>
                <span class="error-text">${errorMsg}</span>
              </div>
            `
          : nothing}

        <form class="login-form" @submit=${handleSubmit}>
          <div class="form-group">
            <label class="form-label" for="login-username">用户名</label>
            <div class="input-wrapper">
              <span class="input-icon">${icons.users}</span>
              <input
                type="text"
                id="login-username"
                class="login-input"
                required
                placeholder="请输入用户名"
                ?disabled=${loading}
              />
            </div>
          </div>

          <div class="form-group">
            <label class="form-label" for="login-password">密码</label>
            <div class="input-wrapper">
              <span class="input-icon">${icons.settings}</span>
              <input
                type="password"
                id="login-password"
                class="login-input"
                required
                placeholder="请输入密码"
                ?disabled=${loading}
              />
            </div>
          </div>

          <button type="submit" class="login-button" ?disabled=${loading}>
            ${loading ? html`<div class="spinner"></div> 正在登录...` : "登录"}
          </button>
        </form>
      </div>
    </div>
  `;
}
