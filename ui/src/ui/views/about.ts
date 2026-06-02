import { html, nothing } from "lit";
import { icons } from "../icons.ts";

export type AboutUninstallMode = "program" | "full";

export type AboutViewProps = {
  basePath: string;
  clearWorkspaceLoading: boolean;
  clearWorkspaceError: string | null;
  onClearWorkspace: () => void | Promise<void>;
  uninstallModalOpen: boolean;
  uninstallMode: AboutUninstallMode;
  uninstallLoading: boolean;
  uninstallError: string | null;
  onOpenUninstallModal: () => void;
  onCloseUninstallModal: () => void;
  onUninstallModeChange: (mode: AboutUninstallMode) => void;
  onConfirmUninstall: () => void | Promise<void>;
};

export function renderAbout(props: AboutViewProps) {
  const logoSrc = props.basePath ? `${props.basePath}/logo_h.png` : "/logo_h.png";

  return html`
    <div class="ops-page ops-about-page" style="padding: 24px; background: var(--bg); color: var(--text-primary); max-width: 1200px; margin: 0 auto;">
      <!-- Header Section with Logo and Branding -->
      <div class="about-header" style="display: flex; align-items: center; gap: 24px; padding: 24px; background: var(--bg-surface, rgba(255,255,255,0.02)); border: 1px solid var(--border); border-radius: 12px; margin-bottom: 24px; box-shadow: 0 4px 20px rgba(0,0,0,0.15);">
        <img src=${logoSrc} alt="ApexOps" style="height: 48px; width: auto; flex-shrink: 0;" />
        <div style="flex: 1;">
          <h1 style="font-size: 20px; font-weight: 700; margin: 0 0 6px 0; color: var(--accent); display: flex; align-items: center; gap: 8px;">
            ApexOps <span style="font-size: 11px; font-weight: 500; background: var(--accent-subtle); color: var(--accent); padding: 2px 8px; border-radius: 20px; text-transform: uppercase; letter-spacing: 0.5px;">v1.0.0 Beta</span>
          </h1>
          <p style="font-size: 13px; color: var(--text-secondary); margin: 0; line-height: 1.5;">
            下一代高智能 AI 自动驾驶运维平台，融合大模型智能诊断与深度健康巡检，保障系统极致稳定与高效运行。
          </p>
        </div>
      </div>

      <!-- Main Layout Grid -->
      <div class="about-grid" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 20px;">
        
        <!-- Left Card: Contact Information -->
        <div class="about-card" style="padding: 24px; background: var(--bg-content); border: 1px solid var(--border); border-radius: 12px; display: flex; flex-direction: column; gap: 16px;">
          <div style="display: flex; align-items: center; gap: 8px; border-bottom: 1px solid var(--border); padding-bottom: 12px; margin-bottom: 4px;">
            <span style="display: flex; width: 16px; height: 16px; color: var(--accent);">${icons.messageSquare}</span>
            <h3 style="font-size: 15px; font-weight: 600; margin: 0; color: var(--text-primary);">联系与合作</h3>
          </div>
          <div>
            <h4 style="font-size: 13px; font-weight: 600; color: var(--text-primary); margin: 0 0 6px 0;">官方支持邮箱</h4>
            <p style="font-size: 12px; color: var(--text-secondary); margin: 0 0 12px 0; line-height: 1.5;">
              如果您在使用过程中遇到任何问题，或者有商业合作意向，欢迎随时与我们取得联系。
            </p>
            <div style="display: flex; align-items: center; gap: 8px;">
              <a href="mailto:sales@aspirecn.com" style="display: inline-flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 500; color: var(--accent); text-decoration: none; padding: 8px 16px; background: var(--accent-subtle); border-radius: 8px; transition: opacity var(--duration-fast); border: 1px solid color-mix(in srgb, var(--accent) 15%, transparent);">
                <span style="display: flex; width: 14px; height: 14px;">${icons.globe}</span>
                sales@aspirecn.com
              </a>
            </div>
          </div>
          <div style="margin-top: auto; border-top: 1px solid var(--border); padding-top: 16px; font-size: 11px; color: var(--text-muted); line-height: 1.6;">
            <p style="margin: 0 0 4px 0;">© 2026 ApexOps. 保留所有权利。</p>
            <p style="margin: 0;">依据相关许可协议，请勿自行替换和修改产品的 Logo 和版权信息。</p>
          </div>
        </div>

        <!-- Right Card: System Operations -->
        <div class="about-card" style="padding: 24px; background: var(--bg-content); border: 1px solid var(--border); border-radius: 12px; display: flex; flex-direction: column; gap: 20px;">
          <div style="display: flex; align-items: center; gap: 8px; border-bottom: 1px solid var(--border); padding-bottom: 12px; margin-bottom: 4px;">
            <span style="display: flex; width: 16px; height: 16px; color: var(--accent);">${icons.settings}</span>
            <h3 style="font-size: 15px; font-weight: 600; margin: 0; color: var(--text-primary);">系统维护与清理</h3>
          </div>

          <!-- Clear Workspace Data -->
          <div style="display: flex; flex-direction: column; gap: 8px;">
            <h4 style="font-size: 13px; font-weight: 600; color: var(--text-primary); margin: 0;">清理工作区文稿与数据</h4>
            <p style="font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5;">
              删除<strong>默认工作区</strong>目录下的全部文件与文件夹（通常为 <code>~/.ApexOps/workspace</code>；Windows 下为 <code>%APPDATA%\\ApexOps\\workspace</code>）。此操作不会影响配置文件和系统其它配置状态。
            </p>
            ${props.clearWorkspaceError
              ? html`<p style="margin: 4px 0 0 0; font-size: 12px; color: #ef4444;" role="alert">${props.clearWorkspaceError}</p>`
              : nothing}
            <div style="margin-top: 4px;">
              <button
                type="button"
                class="btn btn--danger-outline btn--sm"
                style="display: inline-flex; align-items: center; gap: 6px; padding: 8px 12px; border-radius: 6px; font-size: 13px;"
                ?disabled=${props.clearWorkspaceLoading}
                @click=${props.onClearWorkspace}
              >
                <span style="display: flex; width: 14px; height: 14px;">${icons.folder}</span>
                ${props.clearWorkspaceLoading ? html`<span>正在清理…</span>` : html`<span>清理工作区文稿</span>`}
              </button>
            </div>
          </div>

          <!-- Uninstall -->
          <div style="display: flex; flex-direction: column; gap: 8px; border-top: 1px solid var(--border); padding-top: 16px;">
            <h4 style="font-size: 13px; font-weight: 600; color: var(--text-primary); margin: 0;">安全卸载 ApexOps</h4>
            <p style="font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5;">
              在桌面应用或本机网关连接时，可选择仅删除程序或一并清除本地所有数据（配置、缓存、会话等）。
            </p>
            <div style="margin-top: 4px;">
              <button 
                type="button" 
                class="btn btn--danger-outline btn--sm" 
                style="display: inline-flex; align-items: center; gap: 6px; padding: 8px 12px; border-radius: 6px; font-size: 13px;"
                @click=${props.onOpenUninstallModal}
              >
                <span style="display: flex; width: 14px; height: 14px;">${icons.trash}</span>
                安全卸载程序
              </button>
            </div>
          </div>
        </div>

      </div>

      <!-- Uninstall Modal Overlay -->
      ${props.uninstallModalOpen
        ? html`
            <div
              class="modal-overlay"
              role="dialog"
              aria-modal="true"
              aria-labelledby="about-uninstall-title"
              style="position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center; background: rgba(0,0,0,0.6); backdrop-filter: blur(4px);"
              @click=${props.onCloseUninstallModal}
            >
              <div class="modal card about-uninstall-modal" style="width: 100%; max-width: 520px; padding: 28px; background: var(--bg-content); border: 1px solid var(--border); border-radius: 12px; box-shadow: 0 20px 40px rgba(0,0,0,0.4);" @click=${(e: Event) => e.stopPropagation()}>
                <h3 id="about-uninstall-title" class="modal__title" style="font-size: 18px; font-weight: 600; margin: 0 0 8px 0; color: var(--text-primary);">安全卸载 ApexOps</h3>
                <p class="muted small" style="font-size: 12px; color: var(--text-secondary); margin: 0 0 20px 0; line-height: 1.5;">
                  请确认已配置正确的 <strong>Gateway URL</strong> 与 <strong>Token</strong>。卸载任务在主进程退出后由系统脚本彻底清理。
                </p>

                <fieldset class="about-uninstall-fieldset" style="border: none; padding: 0; margin: 0 0 24px 0;">
                  <legend class="visually-hidden">卸载方式</legend>

                  <div class="about-uninstall-options" style="display: flex; flex-direction: column; gap: 12px;">
                    <!-- Option: program only -->
                    <div
                      class="about-uninstall-card ${props.uninstallMode === "program" ? "about-uninstall-card--selected" : ""}"
                      style="padding: 16px; border: 1px solid ${props.uninstallMode === 'program' ? 'var(--accent)' : 'var(--border)'}; background: ${props.uninstallMode === 'program' ? 'var(--accent-subtle)' : 'transparent'}; border-radius: 8px; cursor: pointer; transition: all var(--duration-fast);"
                      @click=${() => props.onUninstallModeChange("program")}
                    >
                      <label class="about-uninstall-mode-label" style="display: flex; align-items: center; gap: 10px; font-weight: 600; font-size: 14px; color: var(--text-primary); cursor: pointer; margin-bottom: 6px;">
                        <input
                          type="radio"
                          name="oo-uninstall-mode"
                          value="program"
                          style="accent-color: var(--accent);"
                          ?checked=${props.uninstallMode === "program"}
                          ?disabled=${props.uninstallLoading}
                          @change=${() => props.onUninstallModeChange("program")}
                        />
                        <span class="about-uninstall-mode-title">仅卸载程序</span>
                      </label>
                      <p style="font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5;">
                        仅删除已安装的应用执行文件，<strong>保留</strong>本地配置文件与工作区数据目录（默认 <code>~/.openocta</code> / <code>%APPDATA%\\openocta</code>）。
                      </p>
                    </div>

                    <!-- Option: full uninstall -->
                    <div
                      class="about-uninstall-card about-uninstall-card--warn ${props.uninstallMode === "full" ? "about-uninstall-card--selected" : ""}"
                      style="padding: 16px; border: 1px solid ${props.uninstallMode === 'full' ? '#ef4444' : 'var(--border)'}; background: ${props.uninstallMode === 'full' ? 'rgba(239, 68, 68, 0.06)' : 'transparent'}; border-radius: 8px; cursor: pointer; transition: all var(--duration-fast);"
                      @click=${() => props.onUninstallModeChange("full")}
                    >
                      <label class="about-uninstall-mode-label" style="display: flex; align-items: center; gap: 10px; font-weight: 600; font-size: 14px; color: var(--text-primary); cursor: pointer; margin-bottom: 6px;">
                        <input
                          type="radio"
                          name="oo-uninstall-mode"
                          value="full"
                          style="accent-color: #ef4444;"
                          ?checked=${props.uninstallMode === "full"}
                          ?disabled=${props.uninstallLoading}
                          @change=${() => props.onUninstallModeChange("full")}
                        />
                        <span class="about-uninstall-mode-title" style="${props.uninstallMode === 'full' ? 'color: #ef4444;' : ''}">全部完全卸载</span>
                      </label>
                      <p style="font-size: 12px; color: var(--text-secondary); margin: 0; line-height: 1.5;">
                        删除应用程序<strong>以及</strong>所有本地状态目录（包括配置、会话日志、历史数据库、缓存及工作区文稿等）。
                      </p>
                      <p class="about-uninstall-note danger" style="font-size: 11px; color: #ef4444; font-weight: 500; margin: 6px 0 0 0;">
                        * 此操作极其危险且不可恢复，请务必确认重要数据已安全备份。
                      </p>
                    </div>
                  </div>
                </fieldset>

                ${props.uninstallError
                  ? html`<p class="about-uninstall-api-error" style="margin: 0 0 16px 0; font-size: 12px; color: #ef4444;" role="alert">${props.uninstallError}</p>`
                  : nothing}

                <div class="modal__actions" style="display: flex; justify-content: flex-end; gap: 12px;">
                  <button
                    type="button"
                    class="btn"
                    style="padding: 8px 16px; border-radius: 6px; font-size: 13px;"
                    ?disabled=${props.uninstallLoading}
                    @click=${props.onCloseUninstallModal}
                  >
                    取消
                  </button>
                  <button
                    type="button"
                    class="btn btn--danger-outline"
                    style="padding: 8px 16px; border-radius: 6px; font-size: 13px; display: inline-flex; align-items: center; gap: 6px;"
                    ?disabled=${props.uninstallLoading}
                    @click=${props.onConfirmUninstall}
                  >
                    ${props.uninstallLoading ? html`<span>正在处理…</span>` : html`<span>确认并开始卸载</span>`}
                  </button>
                </div>
              </div>
            </div>
          `
        : nothing}
    </div>
  `;
}
