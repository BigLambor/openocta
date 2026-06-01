import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { BuiltInProvider } from "./models-builtin.ts";
import { BUILTIN_PROVIDERS, parseModelRef } from "./models-builtin.ts";
import {
  getModelsForProvider,
  getProviderDisplayName,
  providerMatchesSearch,
  renderModelsOverlays,
  type ModelsProps,
  type ModelProvider,
} from "./models.ts";
import { resolveModelProviderLogo } from "./model-provider-logos.js";

export type ModelLibraryCategory = "__all__" | "public" | "local";

export type ModelLibraryProps = ModelsProps & {
  selectedCategory: ModelLibraryCategory;
};

export type ModelLibraryProviderEntry = {
  key: string;
  provider?: ModelProvider;
  builtin?: BuiltInProvider;
  displayName: string;
  baseUrl: string;
  category: Exclude<ModelLibraryCategory, "__all__">;
  modelCount: number;
  previewModel: string | null;
  isDefault: boolean;
};

export type ModelLibraryCategoryInfo = {
  orderedCategories: ModelLibraryCategory[];
  counts: Map<ModelLibraryCategory, number>;
};

const MODEL_LIBRARY_CATEGORIES: ModelLibraryCategory[] = ["__all__", "public", "local"];

function isIpv4Host(hostname: string) {
  const segments = hostname.split(".");
  if (segments.length !== 4) {
    return false;
  }
  return segments.every((segment) => {
    if (!/^\d+$/.test(segment)) {
      return false;
    }
    const value = Number(segment);
    return Number.isInteger(value) && value >= 0 && value <= 255;
  });
}

function isIpv6Host(hostname: string) {
  const normalized = hostname.replace(/^\[/, "").replace(/\]$/, "");
  return normalized.includes(":") && /^[0-9a-f:.]+$/i.test(normalized);
}

export function resolveModelLibraryBaseUrl(provider?: ModelProvider, builtin?: BuiltInProvider) {
  const configuredBaseUrl = provider?.baseUrl?.trim();
  if (configuredBaseUrl) {
    return configuredBaseUrl;
  }
  return builtin?.baseUrl?.trim() ?? "";
}

export function classifyModelLibraryProvider(
  provider?: ModelProvider,
  builtin?: BuiltInProvider,
): Exclude<ModelLibraryCategory, "__all__"> {
  const baseUrl = resolveModelLibraryBaseUrl(provider, builtin);
  if (baseUrl === "(官方)") {
    return "public";
  }
  if (!baseUrl) {
    return "local";
  }
  try {
    const parsed = new URL(baseUrl);
    const hostname = parsed.hostname.trim().toLowerCase();
    if (!hostname || hostname === "localhost" || isIpv4Host(hostname) || isIpv6Host(hostname)) {
      return "local";
    }
    return "public";
  } catch {
    return "local";
  }
}

export function getModelLibraryEntries(
  providers: Record<string, ModelProvider>,
  query: string,
  defaultModelRef: string | null,
): ModelLibraryProviderEntry[] {
  const current = parseModelRef(defaultModelRef);
  const builtinEntries = BUILTIN_PROVIDERS.filter((builtin) =>
    providerMatchesSearch(builtin.id, providers?.[builtin.id], builtin, query),
  ).map((builtin) => {
    const provider = providers?.[builtin.id];
    const models = getModelsForProvider(builtin.id, provider);
    return {
      key: builtin.id,
      provider,
      builtin,
      displayName: getProviderDisplayName(builtin.id, provider),
      baseUrl: resolveModelLibraryBaseUrl(provider, builtin),
      category: classifyModelLibraryProvider(provider, builtin),
      modelCount: models.length,
      previewModel: models[0]?.id ?? null,
      // 只有厂商和模型都匹配时才显示默认模型标识
      isDefault: current?.provider === builtin.id && models.some((m) => m.id === current?.modelId),
    };
  });

  const customEntries = Object.entries(providers ?? {})
    .filter(([key, provider]) => {
      if (BUILTIN_PROVIDERS.some((builtin) => builtin.id === key)) {
        return false;
      }
      return providerMatchesSearch(key, provider, undefined, query);
    })
    .sort(([leftKey, leftProvider], [rightKey, rightProvider]) =>
      getProviderDisplayName(leftKey, leftProvider).localeCompare(
        getProviderDisplayName(rightKey, rightProvider),
        "zh-Hans-CN",
      ),
    )
    .map(([key, provider]) => {
      const models = getModelsForProvider(key, provider);
      return {
        key,
        provider,
        displayName: getProviderDisplayName(key, provider),
        baseUrl: resolveModelLibraryBaseUrl(provider),
        category: classifyModelLibraryProvider(provider),
        modelCount: models.length,
        previewModel: models[0]?.id ?? null,
        // 只有厂商和模型都匹配时才显示默认模型标识
        isDefault: current?.provider === key && models.some((m) => m.id === current?.modelId),
      };
    });

  return [...builtinEntries, ...customEntries];
}

export function computeModelLibraryCategories(
  providers: Record<string, ModelProvider>,
  query: string,
): ModelLibraryCategoryInfo {
  const entries = getModelLibraryEntries(providers, query, null);
  const counts = new Map<ModelLibraryCategory, number>();
  counts.set("__all__", entries.length);
  counts.set("public", entries.filter((entry) => entry.category === "public").length);
  counts.set("local", entries.filter((entry) => entry.category === "local").length);
  return {
    orderedCategories: MODEL_LIBRARY_CATEGORIES,
    counts,
  };
}

function categoryLabel(category: Exclude<ModelLibraryCategory, "__all__">) {
  return category === "public" ? "公有模型" : "本地模型";
}

function renderModelCard(
  entry: ModelLibraryProviderEntry,
  selectedProvider: string | null,
  onSelect: (key: string) => void,
  onUseModelClick: (provider: string) => void,
  onCancelUse: (provider: string) => void,
) {
  const displayBaseUrl = entry.baseUrl || "未配置 Base URL";
  const hasModels = entry.modelCount > 0;
  // 解析 logo
  const logoUrl = resolveModelProviderLogo(
    entry.key,
    entry.displayName,
    entry.baseUrl,
    entry.builtin,
  );
  const logoComponent = logoUrl !== null;

  return html`
    <div class="emp-card-wrap ${selectedProvider === entry.key ? "active" : ""}">
      <div class="emp-card emp-card-btn" @click=${() => onSelect(entry.key)}>
        <div class="emp-card__icon emp-card__icon--default" aria-hidden="true">
          ${logoComponent
            ? html`<img src="${logoUrl}" alt="${entry.displayName}" class="provider-logo" />`
            : icons.modelCube}
        </div>
        <div class="models-provider-actions">
          ${hasModels
            ? entry.isDefault
              ? html`
                <button
                  class="btn btn--sm"
                  @click=${(e: Event) => {
                    e.stopPropagation();
                    onCancelUse(entry.key);
                  }}
                >取消默认</button>
                <span class="market-card-chip market-card-chip--state">默认模型</span>
              `
              : html`
                <button
                  class="btn btn--sm"
                  @click=${(e: Event) => {
                    e.stopPropagation();
                    onUseModelClick(entry.key);
                  }}
                >设为默认</button>
              `
            : nothing}
        </div>
        <h3 class="emp-card__title">${entry.displayName}</h3>
        <p class="emp-card__desc">${displayBaseUrl}</p>
      </div>
    </div>
  `;
}

export function renderModelLibrary(props: ModelLibraryProps) {
  const entries = getModelLibraryEntries(props.providers, props.providerSearchQuery, props.defaultModelRef);
  const selectedCategory = props.selectedCategory ?? "__all__";
  const visibleEntries =
    selectedCategory === "__all__"
      ? entries
      : entries.filter((entry) => entry.category === selectedCategory);
  const publicEntries = visibleEntries.filter((entry) => entry.category === "public");
  const localEntries = visibleEntries.filter((entry) => entry.category === "local");
  const sections =
    selectedCategory === "__all__"
      ? [
          { title: "公有模型", items: publicEntries },
          { title: "本地模型", items: localEntries },
        ]
      : [
          {
            title: categoryLabel(selectedCategory),
            items: visibleEntries,
          },
        ];
  const showToolbarActions = !props.loading || entries.length > 0;

  return html`
    <main class="emp-page">
      <section class="emp-list-wrap">
        <div class="emp-content">
          <div class="emp-main">
            ${showToolbarActions
              ? html`
                  <div class="emp-main__body">
                    <div class="emp-toolbar__actions">
                      <div class="emp-search">
                        <span class="input"><input
                          class="emp-search__input"
                          type="text"
                          placeholder="搜索"
                          .value=${props.providerSearchQuery}
                          ?disabled=${props.loading}
                          @input=${(e: Event) => props.onProviderSearchChange((e.target as HTMLInputElement).value)}
                        /></span>
                        <span class="emp-search__icon" aria-hidden="true">${icons.search}</span>
                      </div>
                      <button class="btn" type="button" ?disabled=${props.loading} @click=${props.onRefresh}>
                        刷新
                      </button>
                      <button class="btn primary" type="button" ?disabled=${props.loading} @click=${props.onAddProvider}>
                        添加
                      </button>
                    </div>
                    <div class="emp-sections">
                      ${sections.map((section) =>
                        section.items.length === 0
                          ? nothing
                          : html`
                              <div class="emp-section">
                                <div class="emp-section__header">
                                  <h3 class="emp-section__title">${section.title}</h3>
                                </div>
                                <div class="emp-grid">
                                  ${section.items.map((entry) =>
                                    renderModelCard(entry, props.selectedProvider, (key) =>
                                      props.onSelect(props.selectedProvider === key ? null : key),
                                      props.onUseModelClick, props.onCancelUse,
                                    ),
                                  )}
                                </div>
                              </div>
                            `,
                      )}
                    </div>
                  </div>
                `
              : nothing}

            ${props.loading
              ? html`<div class="emp-loading">加载中...</div>`
              : visibleEntries.length === 0
                ? html`<div class="emp-empty">暂无匹配的模型厂商</div>`
                : nothing}
          </div>
        </div>
      </section>
    </main>

    ${renderModelsOverlays(props)}
  `;
}
