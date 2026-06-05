import { html, nothing } from "lit";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import type { EmployeeDetail, EmployeeListItem } from "../controllers/remote-market.ts";
import { resolveLogoUrl } from "../controllers/remote-market.ts";
import { groupByCategoryKey } from "../utils/category-helpers.ts";
import { icons } from "../icons.js";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { t } from "../strings.js";
import { toSanitizedMarkdownHtml } from "../markdown.ts";

export type EmployeeMarketProps = {
  loading: boolean;
  error: string | null;
  query: string;
  category: string;
  categoryDescendants: string[];
  items: EmployeeListItem[];
  selectedId: number | string | null;
  selectedDetail: EmployeeDetail | null;
  installedIds?: Set<string>;
  /** 已通过安装接口安装的远程 id 集合 */
  installedRemoteIds?: Set<string>;
  installingId?: string | null;
  onQueryChange: (next: string) => void;
  onCategoryChange: (next: string) => void;
  onRefresh: () => void;
  onSelect: (id: number | string) => void;
  onDetailClose: () => void;
  onAdd?: () => void;
  /** 安装远程数字员工 */
  onInstall?: (id: number | string, category?: string) => Promise<void>;
  /** 删除本地员工（id 为 local:xxx 时的实际 id） */
  onDelete?: (localId: string) => Promise<void>;
  /** 打开会话：跳转到消息页并开始对话（employeeId 为本地员工 id） */
  onOpenEmployee?: (employeeId: string) => void;
  /** 修改本地员工（打开修改弹框，localId 为本地员工 id） */
  onEdit?: (localId: string) => void;
  /** 远程 id -> 本地 id 映射（安装后由后端返回） */
  remoteToLocalMap?: Record<string, string>;
};

export function normalizeCategory(raw?: string) {
  const v = (raw ?? "").trim();
  return v ? v : "其它";
}

function normalizeQuery(raw: string) {
  return (raw ?? "").trim().toLowerCase();
}

export type EmployeeMarketCategoryInfo = {
  orderedCategories: string[];
  counts: Map<string, number>;
};

// TODO: catalog-pages.test.ts 依赖此函数，测试重构后删除
export function computeEmployeeMarketCategories(
  items: EmployeeListItem[],
  query: string
): EmployeeMarketCategoryInfo {
  const q = normalizeQuery(query);
  const filteredByQuery = (items ?? []).filter((it) => {
    if (!q) return true;
    const text = `${it.name ?? ""} ${it.description ?? ""}`.toLowerCase();
    return text.includes(q);
  });
  const counts = new Map<string, number>();
  counts.set("__all__", filteredByQuery.length);
  for (const it of filteredByQuery) {
    const cat = normalizeCategory(it.category);
    counts.set(cat, (counts.get(cat) ?? 0) + 1);
  }
  const orderedCategories = [
    "__all__",
    ...Array.from(counts.keys())
      .filter((k) => k !== "__all__")
      .sort((a, b) => a.localeCompare(b, "zh-Hans-CN")),
  ];
  return { orderedCategories, counts };
}

function splitCsv(raw?: string) {
  if (!raw) return [];
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

const EMP_CARD_TAGS_MAX = 3;

function renderCardTags(tagsRaw?: string, currentCategory?: string) {
  const tags = splitCsv(tagsRaw);
  const isInSingleCategoryView = currentCategory && currentCategory !== "__all__";
  const visibleTags = isInSingleCategoryView
    ? tags.filter((t) => t !== currentCategory)
    : tags;
  if (visibleTags.length === 0) return null;
  const show = visibleTags.slice(0, EMP_CARD_TAGS_MAX);
  const hasMore = visibleTags.length > EMP_CARD_TAGS_MAX;
  return html`
    <div class="market-card-meta">
      ${show.map((t) => html`<span class="market-card-chip">${t}</span>`)}
      ${hasMore ? html`<span class="market-card-chip market-card-chip--muted">...</span>` : nothing}
    </div>
  `;
}

/** Strip YAML frontmatter (--- ... ---) from the start of markdown. */
function stripFrontmatter(text: string): string {
  const trimmed = text.trimStart();
  if (!trimmed.startsWith("---")) return text;
  const afterFirst = trimmed.slice(3);
  const newlineIdx = afterFirst.search(/\r?\n/);
  if (newlineIdx === -1) return text;
  const rest = afterFirst.slice(newlineIdx + (afterFirst[newlineIdx] === "\r" ? 2 : 1));
  const closeMatch = rest.match(/\r?\n\s*---\s*\r?\n?/);
  if (!closeMatch) return text;
  return rest.slice(closeMatch.index! + closeMatch[0].length).trimStart();
}

function statusLabel(status?: string) {
  const v = (status ?? "").trim().toLowerCase();
  if (!v) return "";
  if (v === "open") return "开放";
  if (v === "paid") return "收费";
  if (v === "private") return "私有";
  return status ?? "";
}

function isInstalledEmployee(
  props: EmployeeMarketProps,
  item: EmployeeListItem | EmployeeDetail
) {
  const remoteId = String(item.id);
  const isLocal = typeof item.id === "string" && remoteId.startsWith("local:");
  return isLocal || (props.installedIds?.has(remoteId) ?? false) || (props.installedRemoteIds?.has(remoteId) ?? false);
}

function localEmployeeId(props: EmployeeMarketProps, item: EmployeeListItem | EmployeeDetail) {
  const remoteId = String(item.id);
  const isLocal = typeof item.id === "string" && remoteId.startsWith("local:");
  return isLocal ? remoteId.replace(/^local:/, "") : (props.remoteToLocalMap?.[remoteId] ?? "");
}

function renderEmployeeCardAction(
  props: EmployeeMarketProps,
  item: EmployeeListItem | EmployeeDetail
) {
  const installed = isInstalledEmployee(props, item);
  const remoteId = String(item.id);
  const installing = props.installingId === remoteId;
  if (installed) {
    return nothing;
  }
  if (props.onInstall) {
    return html`
      <button
        class="btn small"
        type="button"
        ?disabled=${installing}
        @click=${(e: Event) => {
          e.stopPropagation();
          void props.onInstall!(item.id, item.category);
        }}
      >
        ${installing ? "安装中" : "安装"}
      </button>
    `;
  }
  return html`
    <a
      class="btn small"
      href=${`/api/v1/employees/${item.id}/download`}
      target="_blank"
      rel="noopener"
      title="下载"
      @click=${(e: Event) => e.stopPropagation()}
    >
      安装
    </a>
  `;
}

function renderEmployeeDetailActions(props: EmployeeMarketProps, detail: EmployeeDetail) {
  const installed = isInstalledEmployee(props, detail);
  const localId = localEmployeeId(props, detail);
  if (installed) {
    return html`
      <div class="market-card-actions">
        ${props.onEdit && localId
          ? html`<button class="btn primary" type="button" @click=${() => props.onEdit!(localId)}>编辑</button>`
          : nothing}
        ${props.onOpenEmployee && localId
          ? html`<button class="btn" type="button" @click=${() => props.onOpenEmployee!(localId)}>开始会话</button>`
          : nothing}
        ${props.onDelete && localId
          ? html`
              <button
                class="btn"
                type="button"
                @click=${async () => {
                  if (!(await nativeConfirm(t("employeeDeleteConfirm")))) {
                    return;
                  }
                  void props.onDelete!(localId);
                }}
              >
                删除
              </button>
            `
          : nothing}
      </div>
    `;
  }
  if (props.onInstall) {
    return html`
      <button
        class="btn primary"
        type="button"
        ?disabled=${props.installingId === String(detail.id)}
        @click=${() => void props.onInstall!(detail.id, detail.category)}
      >
        ${props.installingId === String(detail.id) ? "安装中" : "安装"}
      </button>
    `;
  }
  return html`
    <a
      class="btn primary"
      href=${`/api/v1/employees/${detail.id}/download`}
      target="_blank"
      rel="noopener"
      title="下载"
    >
      安装
    </a>
  `;
}

export function renderEmployeeMarket(props: EmployeeMarketProps) {
  const effectiveCategory = (props.category ?? "").trim() || "__all__";
  const q = normalizeQuery(props.query);
  const filteredByQuery = (props.items ?? []).filter((it) => {
    if (!q) return true;
    const text = `${it.name ?? ""} ${it.description ?? ""}`.toLowerCase();
    return text.includes(q);
  });

  const { filtered: filteredItems, sections: sectionsFixed } = groupByCategoryKey(
    filteredByQuery,
    effectiveCategory,
    props.categoryDescendants,
    (it) => normalizeCategory(it.category)
  );
  const showToolbarActions = !props.error || (props.items?.length ?? 0) > 0;

  const toolbarActions = html`
    <div class="emp-toolbar__actions">
      <div class="emp-search">
        <span class="input"><input
          class="emp-search__input"
          type="text"
          placeholder="搜索"
          .value=${props.query}
          ?disabled=${props.loading}
          @input=${(e: Event) => props.onQueryChange((e.target as HTMLInputElement).value)}
        /></span>
        <span class="emp-search__icon" aria-hidden="true">${icons.search}</span>
      </div>
      <button class="btn" @click=${props.onRefresh} ?disabled=${props.loading}>刷新</button>
      ${props.onAdd ? html`<button class="btn primary" @click=${props.onAdd}>新增</button>` : nothing}
    </div>
  `;

  const installedItems = filteredByQuery.filter((it) => {
    const remoteId = String(it.id);
    const isLocal = typeof it.id === "string" && remoteId.startsWith("local:");
    return (
      isLocal ||
      (props.installedIds?.has(remoteId) ?? false) ||
      (props.installedRemoteIds?.has(remoteId) ?? false)
    );
  });
  const showSections = !props.loading && !(filteredItems.length === 0 && installedItems.length === 0);
  const showMainBody = showToolbarActions || installedItems.length > 0 || showSections;

  return html`
    <main class="emp-page">
      <section class="emp-list-wrap">
        <div class="emp-content">
          <div class="emp-main">
            ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}
            ${showMainBody
              ? html`
                  <div class="emp-main__body">
                    ${showToolbarActions ? toolbarActions : nothing}
                    ${(() => {
                      if (installedItems.length === 0) return nothing;
                      return html`
                        <div class="emp-installed-section">
                          <h3 class="emp-section__title">已安装 (${installedItems.length})</h3>
                          <div class="emp-grid emp-installed-grid">
                            ${installedItems.map((it) => {
                              const selected = props.selectedId === it.id;
                              const logoUrl = resolveLogoUrl(it.logo_url);
                              return html`
                                <div class="emp-card-wrap ${selected ? "active" : ""}">
                                  <div class="emp-card emp-card-btn" @click=${() => props.onSelect(it.id)}>
                                    <div class="emp-card__icon ${!logoUrl ? "emp-card__icon--default" : ""}">
                                      ${logoUrl ? html`<img src=${logoUrl} alt="" />` : html`
                                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                                          <circle cx="12" cy="7" r="4"/>
                                        </svg>
                                      `}
                                    </div>
                                    <div class="emp-card__actions">
                                      ${renderEmployeeCardAction(props, it)}
                                    </div>
                                    <h3 class="emp-card__title">${it.name}</h3>
                                    <p class="emp-card__desc">${it.description ?? "暂无描述"}</p>
                                    ${renderCardTags(it.tags, effectiveCategory)}
                                  </div>
                                </div>
                              `;
                            })}
                          </div>
                        </div>
                      `;
                    })()}
                    ${showSections
                      ? html`
                          <div class="emp-sections">
                            ${sectionsFixed.map(
                              (section) =>
                                section.items.length > 0
                                  ? html`
                                      <div class="emp-section">
                                        <div class="emp-section__header">
                                          <h3 class="emp-section__title">${section.title}</h3>
                                        </div>
                                        <div class="emp-grid">
                                          ${section.items.map((it) => {
                                            const selected = props.selectedId === it.id;
                                            const logoUrl = resolveLogoUrl(it.logo_url);
                                            return html`
                                              <div class="emp-card-wrap ${selected ? "active" : ""}">
                                                <div class="emp-card emp-card-btn" @click=${() => props.onSelect(it.id)}>
                                                  <div class="emp-card__icon ${!logoUrl ? "emp-card__icon--default" : ""}">
                                                    ${logoUrl
                                                      ? html`<img src=${logoUrl} alt="" />`
                                                      : html`
                                                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                                                            <circle cx="12" cy="7" r="4"/>
                                                          </svg>
                                                        `}
                                                  </div>
                                                  <div class="emp-card__actions">
                                                    ${renderEmployeeCardAction(props, it)}
                                                  </div>
                                                  <h3 class="emp-card__title">${it.name}</h3>
                                                  <p class="emp-card__desc">${it.description ?? "暂无描述"}</p>
                                                  ${renderCardTags(it.tags, effectiveCategory)}
                                                </div>
                                              </div>
                                            `;
                                          })}
                                        </div>
                                      </div>
                                    `
                                  : nothing,
                            )}
                          </div>
                        `
                      : nothing}
                  </div>
                `
              : nothing}

            ${props.loading
              ? html`<div class="emp-loading">加载中...</div>`
              : filteredItems.length === 0 && installedItems.length === 0
                ? html`<div class="emp-empty">暂无匹配的助手模板</div>`
                : nothing}
          </div>
        </div>

        ${props.selectedDetail
          ? html`
              <div class="modal-overlay" @click=${props.onDetailClose} role="dialog" aria-modal="true" aria-labelledby="emp-detail-title">
                <div class="modal card emp-detail-modal emp-detail-modal--large" @click=${(e: Event) => e.stopPropagation()}>
                  <div class="emp-detail-modal__header">
                    <div class="emp-detail-header" style="flex: 1; min-width: 0;">
                      <div class="emp-detail-title-wrap">
                        ${(() => {
                          const logoUrl = resolveLogoUrl(props.selectedDetail.logo_url);
                          return logoUrl
                            ? html`<div class="emp-detail-logo"><img src=${logoUrl} alt="" /></div>`
                            : html`
                              <div class="emp-detail-logo emp-detail-logo--default">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                                  <circle cx="12" cy="7" r="4"/>
                                </svg>
                              </div>
                            `;
                        })()}
                        <h1 id="emp-detail-title" class="emp-detail-title">${props.selectedDetail.name}</h1>
                        <div class="emp-detail-tags">
                          <span class="badge ghost">${statusLabel(props.selectedDetail.status)}</span>
                          ${(() => {
                            const detailCategory = normalizeCategory(props.selectedDetail.category);
                            const hideDetailCategory = effectiveCategory && effectiveCategory !== "__all__" && detailCategory === effectiveCategory;
                            return (props.selectedDetail.category ?? "").trim() && !hideDetailCategory
                              ? html`<span class="badge ghost">${detailCategory}</span>`
                              : nothing;
                          })()}
                          ${(() => {
                            const detailCategory = normalizeCategory(props.selectedDetail.category);
                            const detailTags = splitCsv(props.selectedDetail.tags);
                            const visibleDetailTags = detailTags.filter((t) => {
                              const nt = normalizeCategory(t);
                              if (nt === detailCategory) return false;
                              if (effectiveCategory && effectiveCategory !== "__all__" && nt === effectiveCategory) return false;
                              return true;
                            });
                            return visibleDetailTags.length > 0
                              ? visibleDetailTags.map((t) => html`<span class="badge ghost">${t}</span>`)
                              : nothing;
                          })()}
                        </div>
                      </div>
                      <article class="emp-detail-summary">${props.selectedDetail.description ?? ""}</article>
                      <div class="emp-detail-meta-row">
                        ${renderEmployeeDetailActions(props, props.selectedDetail)}
                      </div>
                    </div>
                    <button
                      class="emp-detail-modal__close"
                      type="button"
                      aria-label="关闭"
                      @click=${props.onDetailClose}
                    >
                      ${icons.x}
                    </button>
                  </div>

                  <div class="emp-detail-modal__body">
                    ${props.selectedDetail.readme
                      ? html`<div class="emp-detail-markdown emp-detail-content">${unsafeHTML(toSanitizedMarkdownHtml(stripFrontmatter(props.selectedDetail.readme)))}</div>`
                      : html`<div class="emp-detail-content-empty">无 README</div>`}
                  </div>
                </div>
              </div>
            `
          : nothing}
      </section>
    </main>
  `;
}
