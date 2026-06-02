import { html } from "lit";
import { icons } from "../icons.ts";

export const ONLINE_DOCUMENTATION_URL = "https://databuff.yuque.com/org-wiki-databuff-spr8e6/lqn7on";

export type DocumentationViewProps = {
  url?: string;
  onOpenExternal?: () => void;
};

export function renderDocumentation(props: DocumentationViewProps = {}) {
  return html`
    <div class="ops-page ops-documentation-empty" style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: calc(100vh - 120px); min-height: 400px; padding: 48px; text-align: center;">
      <div class="empty-icon-wrap" style="display: flex; align-items: center; justify-content: center; width: 64px; height: 64px; border-radius: 50%; background: var(--accent-subtle); color: var(--accent); margin-bottom: 20px; box-shadow: 0 4px 12px rgba(var(--accent-rgb), 0.1);">
        <span style="display: flex; width: 32px; height: 32px;">${icons.documentation ?? icons.fileText}</span>
      </div>
      <h2 style="font-size: 18px; font-weight: 600; color: var(--text-primary); margin: 0 0 8px 0;">文档内容待补充</h2>
      <p style="font-size: 14px; color: var(--text-secondary); max-width: 360px; margin: 0; line-height: 1.5;">
        在线文档模块正在整理完善中，更多使用指南及开发规范敬请期待。
      </p>
    </div>
  `;
}
