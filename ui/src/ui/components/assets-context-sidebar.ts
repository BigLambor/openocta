import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import type { IconName } from "../icons.ts";
import { opsDomainIcon } from "./domain-filter.ts";

export type AssetsDomainSidebarItem = {
  key: string;
  label: string;
  count: number;
};

export type AssetsViewSidebarItem<T extends string = string> = {
  id: T;
  label: string;
  icon: IconName;
  count?: number;
  /** 弱化展示：靠后、次要样式 */
  muted?: boolean;
};

export type AssetsContextSidebarProps<T extends string = string> = {
  domains: AssetsDomainSidebarItem[];
  selectedDomain: string;
  views: AssetsViewSidebarItem<T>[];
  activeView: T;
  searchQuery: string;
  statusFilter: string;
  onDomainChange?: (domain: string) => void;
  onViewChange?: (view: T) => void;
  onSearchChange?: (query: string) => void;
  onStatusFilterChange?: (status: string) => void;
};

function renderNavItem(params: {
  isActive: boolean;
  icon: IconName;
  label: string;
  count?: number;
  muted?: boolean;
  onClick: () => void;
}) {
  const { isActive, icon, label, count, muted, onClick } = params;
  return html`
    <button
      type="button"
      class="ops-sidebar-nav-item ${isActive ? "ops-sidebar-nav-item--active" : ""} ${muted ? "ops-sidebar-nav-item--muted" : ""}"
      title=${label}
      @click=${onClick}
    >
      <span class="ops-nav-icon" aria-hidden="true">${icons[icon] ?? icons.folder}</span>
      <span class="ops-sidebar-nav-item__label">${label}</span>
      ${count != null
        ? html`<span class="ops-sidebar-nav-item__badge">${count}</span>`
        : nothing}
    </button>
  `;
}

export function renderAssetsContextSidebar<T extends string>(props: AssetsContextSidebarProps<T>) {
  return html`
    <aside class="ops-sidebar-container assets-context-sidebar">
      <div class="ops-sidebar-section">
        <div class="ops-sidebar-section__label">运维技术域</div>
        <nav class="ops-sidebar-nav">
          ${props.domains.map((item) =>
            renderNavItem({
              isActive: props.selectedDomain === item.key,
              icon: opsDomainIcon(item.key),
              label: item.label,
              count: item.count,
              onClick: () => props.onDomainChange?.(item.key),
            }),
          )}
        </nav>
      </div>

      <div class="ops-sidebar-section">
        <div class="ops-sidebar-section__label">资产分类</div>
        <nav class="ops-sidebar-nav">
          ${props.views.map((item) =>
            renderNavItem({
              isActive: props.activeView === item.id,
              icon: item.icon,
              label: item.label,
              count: item.count,
              muted: item.muted,
              onClick: () => props.onViewChange?.(item.id),
            }),
          )}
        </nav>
      </div>

      <div class="ops-sidebar-filters">
        <div>
          <div class="ops-sidebar-section__label">搜索过滤</div>
          <input
            type="text"
            class="ops-sidebar-filter-input"
            placeholder="搜名称 / 负责人 / 组件..."
            .value=${props.searchQuery}
            @input=${(e: Event) => props.onSearchChange?.((e.target as HTMLInputElement).value)}
          />
        </div>

        <div>
          <div class="ops-sidebar-section__label">资产运行状态</div>
          <select
            class="ops-sidebar-filter-select"
            .value=${props.statusFilter}
            @change=${(e: Event) => props.onStatusFilterChange?.((e.target as HTMLSelectElement).value)}
          >
            <option value="all">全部状态</option>
            <option value="healthy">健康</option>
            <option value="warning">亚健康</option>
            <option value="critical">异常</option>
          </select>
        </div>
      </div>
    </aside>
  `;
}
