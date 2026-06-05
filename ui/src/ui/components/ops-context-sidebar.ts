import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import {
  OPS_DOMAIN_OPTIONS,
  canAccessOpsDomain,
  normalizeOpsDomain,
  opsDomainLabel,
  type OpsDomainKey,
  type DomainFilterUser,
} from "./domain-filter.ts";

export type SidebarItem<T extends string = string> = {
  id: T;
  label: string;
  icon: keyof typeof icons;
  badge?: string | number | null;
};

export type OpsContextSidebarProps<T extends string = string> = {
  selectedDomain: string;
  user?: DomainFilterUser;
  items: SidebarItem<T>[];
  activeItemId: T;
  onItemChange?: (id: T) => void;
  onDomainChange?: (domain: string) => void;
  domainSummary?: {
    alertsCount?: number;
    clustersCount?: number;
    score?: number | null;
  };
  includeAllDomain?: boolean;
};

export function renderOpsContextSidebar<T extends string>(props: OpsContextSidebarProps<T>) {
  // Persistence key for sidebar collapse state
  const storageKey = "openocta.ops-sidebar.collapsed";
  let isCollapsed = false;
  if (typeof localStorage !== "undefined") {
    isCollapsed = localStorage.getItem(storageKey) === "true";
  }

  const selectedDomain = normalizeOpsDomain(props.selectedDomain);
  const user = props.user ?? null;
  const filteredOptions = OPS_DOMAIN_OPTIONS.filter((opt) => {
    if (opt.key === "all" && props.includeAllDomain === false) {
      return false;
    }
    return canAccessOpsDomain(user, opt.key);
  });

  const toggleCollapse = (e: Event) => {
    e.stopPropagation();
    const nextCollapsed = !isCollapsed;
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(storageKey, String(nextCollapsed));
    }
    // Dispatch custom event to trigger parent re-render
    const el = (e.currentTarget as HTMLElement).closest(".ops-sidebar-container");
    if (el) {
      el.dispatchEvent(new CustomEvent("sidebar-toggle", { bubbles: true, composed: true }));
    }
  };

  const selectDomain = (domainKey: string) => {
    props.onDomainChange?.(domainKey);
  };

  return html`
    <aside
      class="ops-sidebar-container ${isCollapsed ? "ops-sidebar--collapsed" : ""}"
      style="display: flex; flex-direction: column; width: ${isCollapsed ? "56px" : "240px"}; background: var(--bg-surface, #fff); border-right: 1px solid var(--border-color, #eee); height: 100%; transition: width 0.2s cubic-bezier(0.4, 0, 0.2, 1); box-sizing: border-box; flex-shrink: 0;"
    >
      <!-- Domain Dropdown / Selector -->
      <div style="padding: 12px; border-bottom: 1px solid var(--border-color, #eee);">
        ${isCollapsed
          ? html`
              <div
                style="width: 32px; height: 32px; border-radius: 4px; display: flex; align-items: center; justify-content: center; background: var(--bg-hover, rgba(0,0,0,0.04)); cursor: pointer;"
                title=${opsDomainLabel(selectedDomain)}
                @click=${() => {
                  const currentIndex = filteredOptions.findIndex(o => o.key === selectedDomain);
                  const nextIndex = (currentIndex + 1) % filteredOptions.length;
                  selectDomain(filteredOptions[nextIndex].key);
                }}
              >
                <span style="font-size: 11px; font-weight: 600;">
                  ${selectedDomain === "all" ? "全" : opsDomainLabel(selectedDomain, true).slice(0, 2)}
                </span>
              </div>
            `
          : html`
              <div style="font-size: 11px; color: var(--text-muted, #777); font-weight: 500; margin-bottom: 6px;">技术上下文</div>
              <div style="position: relative;">
                <select
                  style="width: 100%; padding: 6px 30px 6px 8px; border: 1px solid var(--border-color, #ccc); border-radius: 4px; background: var(--bg-surface, #fff); font-size: 13px; font-weight: 500; cursor: pointer; appearance: none;"
                  .value=${selectedDomain}
                  @change=${(e: Event) => selectDomain((e.target as HTMLSelectElement).value)}
                >
                  ${filteredOptions.map(
                    (opt) => html`<option value=${opt.key}>${opt.label}</option>`,
                  )}
                </select>
                <div style="position: absolute; right: 8px; top: 50%; transform: translateY(-50%); pointer-events: none; opacity: 0.6; display: flex;">
                  ${icons.chevronDown}
                </div>
              </div>
            `}
      </div>

      <!-- Domain Stats Summary (Visible only when expanded and domain selected) -->
      ${!isCollapsed && props.domainSummary
        ? html`
            <div style="padding: 12px; font-size: 12px; display: flex; flex-direction: column; gap: 6px; background: var(--bg-highlight, rgba(0,0,0,0.01)); border-bottom: 1px solid var(--border-color, #eee);">
              ${props.domainSummary.score != null
                ? html`
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                      <span>健康度分数</span>
                      <strong style="color: ${props.domainSummary.score >= 90 ? "#4caf50" : props.domainSummary.score >= 75 ? "#ff9800" : "#f44336"}">
                        ${props.domainSummary.score}分
                      </strong>
                    </div>
                  `
                : nothing}
              ${props.domainSummary.alertsCount !== undefined
                ? html`
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                      <span>活动告警数</span>
                      <strong style="color: ${props.domainSummary.alertsCount > 0 ? "#f44336" : "inherit"}">
                        ${props.domainSummary.alertsCount} 个
                      </strong>
                    </div>
                  `
                : nothing}
              ${props.domainSummary.clustersCount !== undefined
                ? html`
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                      <span>纳管集群数</span>
                      <strong>${props.domainSummary.clustersCount} 个</strong>
                    </div>
                  `
                : nothing}
            </div>
          `
        : nothing}

      <!-- Sidebar Nav Items -->
      <nav style="flex: 1; overflow-y: auto; padding: 12px 6px; display: flex; flex-direction: column; gap: 4px;">
        ${props.items.map((item) => {
          const isActive = props.activeItemId === item.id;
          return html`
            <button
              type="button"
              class="ops-sidebar-item ${isActive ? "ops-sidebar-item--active" : ""}"
              style="display: flex; align-items: center; width: 100%; border: none; padding: 8px 10px; border-radius: 4px; cursor: pointer; text-align: left; transition: all 0.15s ease; font-size: 13px; font-weight: ${isActive ? "600" : "500"}; background: ${isActive ? "var(--bg-active, rgba(33,150,243,0.08))" : "transparent"}; color: ${isActive ? "var(--primary-color, #2196f3)" : "var(--text-color, #333)"};"
              title=${item.label}
              @click=${() => props.onItemChange?.(item.id)}
            >
              <span style="display: flex; align-items: center; justify-content: center; width: 20px; margin-right: ${isCollapsed ? "0" : "10px"}; opacity: ${isActive ? 1 : 0.7}; flex-shrink: 0;">
                ${icons[item.icon] ?? icons.folder}
              </span>
              ${!isCollapsed
                ? html`
                    <span style="flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${item.label}</span>
                    ${item.badge != null && item.badge !== 0
                      ? html`
                          <span
                            style="font-size: 10px; font-weight: bold; padding: 2px 6px; border-radius: 10px; background: ${isActive ? "var(--primary-color, #2196f3)" : "var(--border-color, #ccc)"}; color: #fff; margin-left: 6px;"
                          >
                            ${item.badge}
                          </span>
                        `
                      : nothing}
                  `
                : nothing}
            </button>
          `;
        })}
      </nav>

      <!-- Toggle Collapse Button at bottom -->
      <div style="padding: 10px; border-top: 1px solid var(--border-color, #eee); display: flex; justify-content: ${isCollapsed ? "center" : "flex-end"};">
        <button
          class="ops-btn ops-btn--ghost"
          type="button"
          style="padding: 6px; border-radius: 4px;"
          title=${isCollapsed ? "展开侧栏" : "收起侧栏"}
          @click=${toggleCollapse}
        >
          <span style="display: flex;">
            ${isCollapsed ? icons.chevronRight : icons.chevronLeft}
          </span>
        </button>
      </div>
    </aside>
  `;
}
