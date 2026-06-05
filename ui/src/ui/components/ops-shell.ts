import { html, nothing, type TemplateResult } from "lit";
import { icons } from "../icons.ts";

export type OpsShellHeaderProps = {
  kicker: string;
  title: string;
  description?: string;
  toolbar?: TemplateResult;
};

export function renderOpsShellHeader(props: OpsShellHeaderProps) {
  return html`
    <header class="ops-shell-header">
      <div class="ops-shell-header__text">
        <div class="ops-shell-kicker">${props.kicker}</div>
        <h1>${props.title}</h1>
        ${props.description
          ? html`<p class="ops-shell-header__desc">${props.description}</p>`
          : nothing}
      </div>
      ${props.toolbar
        ? html`<div class="ops-shell-toolbar">${props.toolbar}</div>`
        : nothing}
    </header>
  `;
}

export type OpsViewNavItem<T extends string = string> = {
  id: T;
  label: string;
  icon: keyof typeof icons;
};

export function renderOpsViewNav<T extends string>(
  items: OpsViewNavItem<T>[],
  activeId: T,
  onChange?: (id: T) => void,
) {
  return html`
    <nav class="ops-view-nav" aria-label="子模块导航">
      ${items.map(
        (item) => html`
          <button
            type="button"
            class="ops-view-nav__item ${activeId === item.id ? "ops-view-nav__item--active" : ""}"
            @click=${() => onChange?.(item.id)}
          >
            <span class="ops-view-nav__icon" aria-hidden="true">${icons[item.icon] ?? icons.folder}</span>
            <span>${item.label}</span>
          </button>
        `,
      )}
    </nav>
  `;
}

export type OpsShellStat = {
  label: string;
  value: string | number;
  hint?: string;
  tone?: "blue" | "ok" | "warn" | "danger" | "info";
  icon?: keyof typeof icons;
};

export function renderOpsShellStatGrid(stats: OpsShellStat[]) {
  const toneClass = (tone?: OpsShellStat["tone"]) => {
    switch (tone) {
      case "warn":
        return "stat-icon-slot--warn";
      case "danger":
        return "stat-icon-slot--danger";
      case "ok":
        return "stat-icon-slot--ok";
      default:
        return "stat-icon-slot--blue";
    }
  };
  const valueClass = (tone?: OpsShellStat["tone"]) => {
    switch (tone) {
      case "warn":
        return "warning";
      case "danger":
        return "critical";
      case "ok":
        return "ok";
      default:
        return tone === "info" ? "info" : "";
    }
  };
  return html`
    <section class="stats-grid ops-summary-cards" aria-label="指标概览">
      ${stats.map(
        (stat) => html`
          <article class="stat-card">
            <div class="stat-icon-slot ${toneClass(stat.tone)}">
              ${stat.icon ? icons[stat.icon] : icons.overviewGrid}
            </div>
            <div class="stat-body">
              <h3>${stat.label}</h3>
              <div class="stat-value ${valueClass(stat.tone)}">${stat.value}</div>
              ${stat.hint ? html`<p class="muted" style="margin:4px 0 0;font-size:12px;">${stat.hint}</p>` : nothing}
            </div>
          </article>
        `,
      )}
    </section>
  `;
}
