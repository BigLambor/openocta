import { html, nothing, type TemplateResult } from "lit";
import { icons, type IconName } from "../icons.ts";

export type OpsEmptyProps = {
  icon?: IconName;
  title: string;
  description?: string;
  hint?: string;
  actionLabel?: string;
  onAction?: () => void;
  compact?: boolean;
  /** 左对齐、铺满面板（运维大屏等桌面页） */
  spread?: boolean;
};

export function renderOpsEmpty(props: OpsEmptyProps): TemplateResult {
  const iconName = props.icon ?? "folder";
  return html`
    <div
      class="ops-status ops-status--empty ${props.compact ? "ops-status--compact" : ""} ${props.spread ? "ops-status--spread" : ""}"
    >
      <div class="ops-status__icon" aria-hidden="true">${icons[iconName]}</div>
      <div class="ops-status__title">${props.title}</div>
      ${props.description
        ? html`<p class="ops-status__desc">${props.description}</p>`
        : nothing}
      ${props.hint ? html`<p class="ops-status__hint">${props.hint}</p>` : nothing}
      ${props.actionLabel && props.onAction
        ? html`
            <button type="button" class="ops-status__action" @click=${props.onAction}>
              ${props.actionLabel}
            </button>
          `
        : nothing}
    </div>
  `;
}

export type OpsErrorProps = {
  title?: string;
  message: string;
  onRetry?: () => void;
};

export function renderOpsError(props: OpsErrorProps): TemplateResult {
  return html`
    <div class="ops-status ops-status--error">
      <div class="ops-status__icon" aria-hidden="true">${icons.alertTriangle}</div>
      <div class="ops-status__title">${props.title ?? "加载失败"}</div>
      <p class="ops-status__desc">${props.message}</p>
      ${props.onRetry
        ? html`
            <button type="button" class="ops-status__action ops-status__action--secondary" @click=${props.onRetry}>
              重试
            </button>
          `
        : nothing}
    </div>
  `;
}

export type OpsSkeletonProps = {
  lines?: number;
};

export function renderOpsSkeleton(props: OpsSkeletonProps = {}): TemplateResult {
  const lines = props.lines ?? 3;
  return html`
    <div class="ops-status ops-status--skeleton" aria-busy="true" aria-label="加载中">
      ${Array.from({ length: lines }, (_, i) => html`
        <div class="ops-skeleton-line" style="width: ${i === lines - 1 ? "60%" : "100%"}"></div>
      `)}
    </div>
  `;
}
