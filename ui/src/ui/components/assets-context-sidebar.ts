import { html } from "lit";
import { icons } from "../icons.ts";
import type { IconName } from "../icons.ts";

export type AssetsDomainSidebarItem = {
  key: string;
  label: string;
  count: number;
};

export type AssetsViewSidebarItem<T extends string = string> = {
  id: T;
  label: string;
  icon: IconName;
  count: number;
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

export function renderAssetsContextSidebar<T extends string>(props: AssetsContextSidebarProps<T>) {
  return html`
    <aside
      class="ops-sidebar-container assets-context-sidebar"
      style="width: 240px; background: var(--bg-surface, #fff); border-right: 1px solid var(--border-color, #eee); height: 100%; display: flex; flex-direction: column; flex-shrink: 0; box-sizing: border-box; padding: 12px 10px; gap: 16px;"
    >
      <div>
        <div style="font-size: 11px; color: var(--text-muted, #777); font-weight: bold; text-transform: uppercase; margin-bottom: 6px; letter-spacing: 0.5px;">运维技术域</div>
        <div style="display: flex; flex-direction: column; gap: 2px;">
          ${props.domains.map((item) => {
            const isActive = props.selectedDomain === item.key;
            return html`
              <button
                type="button"
                style="display: flex; justify-content: space-between; align-items: center; border: none; padding: 6px 10px; border-radius: 4px; font-size: 12px; font-weight: ${isActive ? "600" : "500"}; background: ${isActive ? "var(--bg-active, rgba(33,150,243,0.08))" : "transparent"}; color: ${isActive ? "var(--primary-color, #2196f3)" : "var(--text-color, #333)"}; cursor: pointer; text-align: left; transition: all 0.15s ease;"
                @click=${() => props.onDomainChange?.(item.key)}
              >
                <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; margin-right: 8px;">${item.label}</span>
                <span style="font-size: 10px; background: rgba(0,0,0,0.04); padding: 1px 5px; border-radius: 8px; color: #666; font-weight: 500;">${item.count}</span>
              </button>
            `;
          })}
        </div>
      </div>

      <div>
        <div style="font-size: 11px; color: var(--text-muted, #777); font-weight: bold; text-transform: uppercase; margin-bottom: 6px; letter-spacing: 0.5px;">资产分类</div>
        <div style="display: flex; flex-direction: column; gap: 2px;">
          ${props.views.map((item) => {
            const isActive = props.activeView === item.id;
            return html`
              <button
                type="button"
                style="display: flex; align-items: center; border: none; padding: 6px 10px; border-radius: 4px; font-size: 12px; font-weight: ${isActive ? "600" : "500"}; background: ${isActive ? "var(--bg-active, rgba(33,150,243,0.08))" : "transparent"}; color: ${isActive ? "var(--primary-color, #2196f3)" : "var(--text-color, #333)"}; cursor: pointer; text-align: left; transition: all 0.15s ease;"
                @click=${() => props.onViewChange?.(item.id)}
              >
                <span style="margin-right: 8px; display: flex; opacity: 0.7; font-size: 14px;">${icons[item.icon] ?? icons.folder}</span>
                <span style="flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${item.label}</span>
                <span style="font-size: 10px; opacity: 0.6; font-weight: bold;">${item.count}</span>
              </button>
            `;
          })}
        </div>
      </div>

      <div style="margin-top: auto; display: flex; flex-direction: column; gap: 10px; border-top: 1px solid var(--border-color, #eee); padding-top: 12px;">
        <div>
          <div style="font-size: 11px; color: var(--text-muted, #777); font-weight: bold; text-transform: uppercase; margin-bottom: 4px;">搜索过滤</div>
          <input
            type="text"
            placeholder="搜名称 / 负责人 / 组件..."
            style="width: 100%; padding: 6px 8px; font-size: 12px; border: 1px solid var(--border-color, #ccc); border-radius: 4px; background: var(--bg-surface, #fff); box-sizing: border-box;"
            .value=${props.searchQuery}
            @input=${(e: Event) => props.onSearchChange?.((e.target as HTMLInputElement).value)}
          />
        </div>

        <div>
          <div style="font-size: 11px; color: var(--text-muted, #777); font-weight: bold; text-transform: uppercase; margin-bottom: 4px;">资产运行状态</div>
          <select
            style="width: 100%; padding: 6px; font-size: 12px; border: 1px solid var(--border-color, #ccc); border-radius: 4px; background: var(--bg-surface, #fff); cursor: pointer;"
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
