import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { ConfigUiHints } from "../types.ts";
import { getConfigFieldHelp } from "../config-field-help.js";
import { getConfigFieldLabel } from "../config-field-labels.js";
import { t } from "../strings.js";
import { hintForPath, humanize, schemaType, type JsonSchema } from "./config-form.shared.ts";
import { analyzeConfigSchema, renderConfigForm, getSectionMeta } from "./config-form.ts";

export type ConfigProps = {
  raw: string;
  originalRaw: string;
  valid: boolean | null;
  issues: unknown[];
  loading: boolean;
  saving: boolean;
  applying: boolean;
  updating: boolean;
  connected: boolean;
  schema: unknown;
  schemaLoading: boolean;
  uiHints: ConfigUiHints;
  formMode: "form" | "raw";
  formValue: Record<string, unknown> | null;
  originalValue: Record<string, unknown> | null;
  searchQuery: string;
  activeSection: string | null;
  activeSubsection: string | null;
  onRawChange: (next: string) => void;
  onFormModeChange: (mode: "form" | "raw") => void;
  onFormPatch: (path: Array<string | number>, value: unknown) => void;
  onSearchChange: (query: string) => void;
  onSectionChange: (section: string | null) => void;
  onSubsectionChange: (section: string | null) => void;
  onReload: () => void;
  onSave: () => void;
  onApply: () => void;
  onUpdate: () => void;
  rbacUsersList?: any[];
  rbacRolesList?: any[];
  rbacUsersLoading?: boolean;
  rbacToken?: string | null;
  rbacUser?: any | null;
  onCreateUser?: (username: string, password: string, roleId: number) => Promise<void>;
  onDeleteUser?: (id: number) => Promise<void>;
};

// SVG Icons for sidebar (Lucide-style)
const sidebarIcons = {
  all: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="3" y="3" width="7" height="7"></rect>
      <rect x="14" y="3" width="7" height="7"></rect>
      <rect x="14" y="14" width="7" height="7"></rect>
      <rect x="3" y="14" width="7" height="7"></rect>
    </svg>
  `,
  env: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="3"></circle>
      <path
        d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
      ></path>
    </svg>
  `,
  update: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
      <polyline points="7 10 12 15 17 10"></polyline>
      <line x1="12" y1="15" x2="12" y2="3"></line>
    </svg>
  `,
  agents: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path
        d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z"
      ></path>
      <circle cx="8" cy="14" r="1"></circle>
      <circle cx="16" cy="14" r="1"></circle>
    </svg>
  `,
  auth: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
      <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
    </svg>
  `,
  channels: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
    </svg>
  `,
  messages: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path>
      <polyline points="22,6 12,13 2,6"></polyline>
    </svg>
  `,
  commands: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polyline points="4 17 10 11 4 5"></polyline>
      <line x1="12" y1="19" x2="20" y2="19"></line>
    </svg>
  `,
  hooks: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
    </svg>
  `,
  skills: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polygon
        points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"
      ></polygon>
    </svg>
  `,
  tools: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path
        d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"
      ></path>
    </svg>
  `,
  gateway: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"></circle>
      <line x1="2" y1="12" x2="22" y2="12"></line>
      <path
        d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
      ></path>
    </svg>
  `,
  wizard: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M15 4V2"></path>
      <path d="M15 16v-2"></path>
      <path d="M8 9h2"></path>
      <path d="M20 9h2"></path>
      <path d="M17.8 11.8 19 13"></path>
      <path d="M15 9h0"></path>
      <path d="M17.8 6.2 19 5"></path>
      <path d="m3 21 9-9"></path>
      <path d="M12.2 6.2 11 5"></path>
    </svg>
  `,
  // Additional sections
  meta: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M12 20h9"></path>
      <path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path>
    </svg>
  `,
  logging: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
      <polyline points="14 2 14 8 20 8"></polyline>
      <line x1="16" y1="13" x2="8" y2="13"></line>
      <line x1="16" y1="17" x2="8" y2="17"></line>
      <polyline points="10 9 9 9 8 9"></polyline>
    </svg>
  `,
  browser: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"></circle>
      <circle cx="12" cy="12" r="4"></circle>
      <line x1="21.17" y1="8" x2="12" y2="8"></line>
      <line x1="3.95" y1="6.06" x2="8.54" y2="14"></line>
      <line x1="10.88" y1="21.94" x2="15.46" y2="14"></line>
    </svg>
  `,
  ui: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
      <line x1="3" y1="9" x2="21" y2="9"></line>
      <line x1="9" y1="21" x2="9" y2="9"></line>
    </svg>
  `,
  models: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path
        d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"
      ></path>
      <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
      <line x1="12" y1="22.08" x2="12" y2="12"></line>
    </svg>
  `,
  bindings: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect>
      <rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect>
      <line x1="6" y1="6" x2="6.01" y2="6"></line>
      <line x1="6" y1="18" x2="6.01" y2="18"></line>
    </svg>
  `,
  broadcast: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"></path>
      <path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5"></path>
      <circle cx="12" cy="12" r="2"></circle>
      <path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5"></path>
      <path d="M19.1 4.9C23 8.8 23 15.1 19.1 19"></path>
    </svg>
  `,
  audio: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M9 18V5l12-2v13"></path>
      <circle cx="6" cy="18" r="3"></circle>
      <circle cx="18" cy="16" r="3"></circle>
    </svg>
  `,
  session: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
      <circle cx="9" cy="7" r="4"></circle>
      <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
    </svg>
  `,
  cron: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"></circle>
      <polyline points="12 6 12 12 16 14"></polyline>
    </svg>
  `,
  web: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"></circle>
      <line x1="2" y1="12" x2="22" y2="12"></line>
      <path
        d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
      ></path>
    </svg>
  `,
  discovery: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="11" cy="11" r="8"></circle>
      <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
    </svg>
  `,
  canvasHost: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
      <circle cx="8.5" cy="8.5" r="1.5"></circle>
      <polyline points="21 15 16 10 5 21"></polyline>
    </svg>
  `,
  talk: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
      <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
      <line x1="12" y1="19" x2="12" y2="23"></line>
      <line x1="8" y1="23" x2="16" y2="23"></line>
    </svg>
  `,
  plugins: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M12 2v6"></path>
      <path d="m4.93 10.93 4.24 4.24"></path>
      <path d="M2 12h6"></path>
      <path d="m4.93 13.07 4.24-4.24"></path>
      <path d="M12 22v-6"></path>
      <path d="m19.07 13.07-4.24-4.24"></path>
      <path d="M22 12h-6"></path>
      <path d="m19.07 10.93-4.24 4.24"></path>
    </svg>
  `,
  default: html`
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
      <polyline points="14 2 14 8 20 8"></polyline>
    </svg>
  `,
};

// Section definitions (labels from i18n)
function getSections(): Array<{ key: string; label: string }> {
  return [
    { key: "env", label: t("configEnv") },
    { key: "update", label: t("configUpdate") },
    { key: "agents", label: t("configAgents") },
    { key: "auth", label: t("configAuth") },
    { key: "channels", label: t("configChannels") },
    { key: "messages", label: t("configMessages") },
    { key: "commands", label: t("configCommands") },
    { key: "hooks", label: t("configHooks") },
    { key: "skills", label: t("configSkills") },
    { key: "tools", label: t("configTools") },
    { key: "gateway", label: t("configGateway") },
    { key: "wizard", label: t("configWizard") },
    { key: "meta", label: t("configMeta") },
    { key: "logging", label: t("configLogging") },
    { key: "browser", label: t("configBrowser") },
    { key: "ui", label: t("configUi") },
    { key: "models", label: t("configModels") },
    { key: "bindings", label: t("configBindings") },
    { key: "broadcast", label: t("configBroadcast") },
    { key: "audio", label: t("configAudio") },
    { key: "session", label: t("configSession") },
    { key: "cron", label: t("configCron") },
    { key: "web", label: t("configWeb") },
    { key: "discovery", label: t("configDiscovery") },
    { key: "canvasHost", label: t("configCanvasHost") },
    { key: "talk", label: t("configTalk") },
    { key: "plugins", label: t("configPlugins") },
  ];
}

type SubsectionEntry = {
  key: string;
  label: string;
  description?: string;
  order: number;
};

const ALL_SUBSECTION = "__all__";

function getSectionIcon(key: string) {
  if (key === "rbac") return sidebarIcons.auth;
  return sidebarIcons[key as keyof typeof sidebarIcons] ?? sidebarIcons.default;
}

function resolveSectionMeta(
  key: string,
  schema?: JsonSchema,
): {
  label: string;
  description?: string;
} {
  const meta = getSectionMeta(key);
  if (meta) {
    return meta;
  }
  return {
    label: schema?.title ?? humanize(key),
    description: schema?.description ?? "",
  };
}

function resolveSubsections(params: {
  key: string;
  schema: JsonSchema | undefined;
  uiHints: ConfigUiHints;
}): SubsectionEntry[] {
  const { key, schema, uiHints } = params;
  if (!schema || schemaType(schema) !== "object" || !schema.properties) {
    return [];
  }
  const entries = Object.entries(schema.properties).map(([subKey, node]) => {
    const hint = hintForPath([key, subKey], uiHints);
    const label = getConfigFieldLabel([key, subKey], hint?.label ?? node.title ?? humanize(subKey));
    const description = getConfigFieldHelp([key, subKey], hint?.help ?? node.description ?? "");
    const order = hint?.order ?? 50;
    return { key: subKey, label, description, order };
  });
  entries.sort((a, b) => (a.order !== b.order ? a.order - b.order : a.key.localeCompare(b.key)));
  return entries;
}

function computeDiff(
  original: Record<string, unknown> | null,
  current: Record<string, unknown> | null,
): Array<{ path: string; from: unknown; to: unknown }> {
  if (!original || !current) {
    return [];
  }
  const changes: Array<{ path: string; from: unknown; to: unknown }> = [];

  function compare(orig: unknown, curr: unknown, path: string) {
    if (orig === curr) {
      return;
    }
    if (typeof orig !== typeof curr) {
      changes.push({ path, from: orig, to: curr });
      return;
    }
    if (typeof orig !== "object" || orig === null || curr === null) {
      if (orig !== curr) {
        changes.push({ path, from: orig, to: curr });
      }
      return;
    }
    if (Array.isArray(orig) && Array.isArray(curr)) {
      if (JSON.stringify(orig) !== JSON.stringify(curr)) {
        changes.push({ path, from: orig, to: curr });
      }
      return;
    }
    const origObj = orig as Record<string, unknown>;
    const currObj = curr as Record<string, unknown>;
    const allKeys = new Set([...Object.keys(origObj), ...Object.keys(currObj)]);
    for (const key of allKeys) {
      compare(origObj[key], currObj[key], path ? `${path}.${key}` : key);
    }
  }

  compare(original, current, "");
  return changes;
}

function truncateValue(value: unknown, maxLen = 40): string {
  let str: string;
  try {
    const json = JSON.stringify(value);
    str = json ?? String(value);
  } catch {
    str = String(value);
  }
  if (str.length <= maxLen) {
    return str;
  }
  return str.slice(0, maxLen - 3) + "...";
}

export function renderConfig(props: ConfigProps) {
  const validity = props.valid == null ? "unknown" : props.valid ? "valid" : "invalid";
  const analysis = analyzeConfigSchema(props.schema);
  const formUnsafe = analysis.schema ? analysis.unsupportedPaths.length > 0 : false;

  // Get available sections from schema
  const schemaProps = analysis.schema?.properties ?? {};
  const availableSections = getSections().filter((s) => s.key in schemaProps);

  // Add any sections in schema but not in our list
  const knownKeys = new Set(getSections().map((s) => s.key));
  const extraSections = Object.keys(schemaProps)
    .filter((k) => !knownKeys.has(k))
    .map((k) => ({ key: k, label: k.charAt(0).toUpperCase() + k.slice(1) }));

  const allSections = [...availableSections, ...extraSections];
  if (props.rbacUser?.roleName === "admin") {
    allSections.push({ key: "rbac", label: "用户与权限管理" });
  }

  const activeSectionSchema =
    props.activeSection && analysis.schema && schemaType(analysis.schema) === "object"
      ? analysis.schema.properties?.[props.activeSection]
      : undefined;
  const activeSectionMeta = props.activeSection
    ? resolveSectionMeta(props.activeSection, activeSectionSchema)
    : null;
  const subsections = props.activeSection
    ? resolveSubsections({
        key: props.activeSection,
        schema: activeSectionSchema,
        uiHints: props.uiHints,
      })
    : [];
  const allowSubnav =
    props.formMode === "form" && Boolean(props.activeSection) && subsections.length > 0;
  const isAllSubsection = props.activeSubsection === ALL_SUBSECTION;
  const effectiveSubsection = props.searchQuery
    ? null
    : isAllSubsection
      ? null
      : (props.activeSubsection ?? subsections[0]?.key ?? null);

  // Compute diff for showing changes (works for both form and raw modes)
  const diff = props.formMode === "form" ? computeDiff(props.originalValue, props.formValue) : [];
  const hasRawChanges = props.formMode === "raw" && props.raw !== props.originalRaw;
  const hasChanges = props.formMode === "form" ? diff.length > 0 : hasRawChanges;

  // Save/apply buttons require actual changes to be enabled.
  // Note: formUnsafe warns about unsupported schema paths but shouldn't block saving.
  const canSaveForm = Boolean(props.formValue) && !props.loading && Boolean(analysis.schema);
  const canSave =
    props.connected &&
    !props.saving &&
    hasChanges &&
    (props.formMode === "raw" ? true : canSaveForm);
  const canApply =
    props.connected &&
    !props.applying &&
    !props.updating &&
    hasChanges &&
    (props.formMode === "raw" ? true : canSaveForm);
  const canUpdate = props.connected && !props.applying && !props.updating;

  return html`
    <div class="config-layout">
      <!-- Sidebar -->
      <aside class="config-sidebar">
        <div class="config-sidebar__header">
          <div class="config-sidebar__title">${t("configSettingsTitle")}</div>
          <span
            class="pill pill--sm ${
              validity === "valid" ? "pill--accent" : validity === "invalid" ? "pill--danger" : ""
            }"
            >${
              validity === "valid"
                ? t("configValidityValid")
                : validity === "invalid"
                  ? t("configValidityInvalid")
                  : t("configValidityUnknown")
            }</span
          >
        </div>

        <!-- Search -->
        <div class="config-search">
          <svg
            class="config-search__icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="11" cy="11" r="8"></circle>
            <path d="M21 21l-4.35-4.35"></path>
          </svg>
          <span class="input"><input
            type="text"
            class="config-search__input"
            placeholder=${t("configSearchPlaceholder")}
            .value=${props.searchQuery}
            @input=${(e: Event) => props.onSearchChange((e.target as HTMLInputElement).value)}
          /></span>
          ${
            props.searchQuery
              ? html`
                <button
                  class="config-search__clear"
                  type="button"
                  aria-label="清空搜索"
                  @click=${() => props.onSearchChange("")}
                >
                  ${icons.x}
                </button>
              `
              : nothing
          }
        </div>

        <!-- Section nav -->
        <nav class="config-nav">
          <button
            class="config-nav__item ${props.activeSection === null ? "active" : ""}"
            @click=${() => props.onSectionChange(null)}
          >
            <span class="config-nav__icon">${sidebarIcons.all}</span>
            <span class="config-nav__label">${t("configAllSettings")}</span>
          </button>
          ${allSections.map(
            (section) => html`
              <button
                class="config-nav__item ${props.activeSection === section.key ? "active" : ""}"
                @click=${() => props.onSectionChange(section.key)}
              >
                <span class="config-nav__icon"
                  >${getSectionIcon(section.key)}</span
                >
                <span class="config-nav__label">${section.label}</span>
              </button>
            `,
          )}
        </nav>
      </aside>

      <!-- Main content -->
      <main class="config-main">
        <!-- Action bar -->
        <div class="config-actions">
          <div class="config-actions__left">
            ${
              hasChanges
                ? html`
                  <span class="config-changes-badge"
                    >${
                      props.formMode === "raw"
                        ? t("configUnsavedChanges")
                        : diff.length === 1
                          ? t("configOneUnsavedChange")
                          : `${diff.length} ${t("configUnsavedChangesLabel")}`
                    }</span
                  >
                `
                : html`
                    <span class="config-status muted">${t("configNoChanges")}</span>
                  `
            }
          </div>
          <div class="config-actions__right">
            <button
              class="btn btn--sm"
              ?disabled=${props.loading}
              @click=${props.onReload}
            >
              ${props.loading ? t("commonLoading") : t("commonReload")}
            </button>
            <button
              class="btn btn--sm primary"
              ?disabled=${!canSave}
              @click=${props.onSave}
            >
              ${props.saving ? t("commonSaving") : t("commonSave")}
            </button>
            <button
              class="btn btn--sm"
              ?disabled=${!canApply}
              @click=${props.onApply}
            >
              ${props.applying ? t("configApplying") : t("configApply")}
            </button>
            <button
              class="btn btn--sm"
              ?disabled=${!canUpdate}
              @click=${props.onUpdate}
            >
              ${props.updating ? t("configUpdating") : t("configUpdateButton")}
            </button>
          </div>
        </div>

        <!-- Diff panel (form mode only - raw mode doesn't have granular diff) -->
        ${
          hasChanges && props.formMode === "form"
            ? html`
              <details class="config-diff">
                <summary class="config-diff__summary">
                  <span
                    >${t("configViewPrefix")}${diff.length}
                    ${diff.length === 1 ? t("configPendingChange") : t("configPendingChanges")}</span
                  >
                  <svg
                    class="config-diff__chevron"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </summary>
                <div class="config-diff__content">
                  ${diff.map(
                    (change) => html`
                      <div class="config-diff__item">
                        <div class="config-diff__path">${change.path}</div>
                        <div class="config-diff__values">
                          <span class="config-diff__from"
                            >${truncateValue(change.from)}</span
                          >
                          <span class="config-diff__arrow">→</span>
                          <span class="config-diff__to"
                            >${truncateValue(change.to)}</span
                          >
                        </div>
                      </div>
                    `,
                  )}
                </div>
              </details>
            `
            : nothing
        }
        ${
          activeSectionMeta && props.formMode === "form"
            ? html`
              <div class="config-section-hero">
                <div class="config-section-hero__icon">
                  ${getSectionIcon(props.activeSection ?? "")}
                </div>
                <div class="config-section-hero__text">
                  <div class="config-section-hero__title">
                    ${activeSectionMeta.label}
                  </div>
                  ${
                    activeSectionMeta.description
                      ? html`<div class="config-section-hero__desc">
                        ${activeSectionMeta.description}
                      </div>`
                      : nothing
                  }
                </div>
              </div>
            `
            : nothing
        }
        ${
          allowSubnav
            ? html`
              <div class="config-subnav">
                <button
                  class="config-subnav__item ${effectiveSubsection === null ? "active" : ""}"
                  @click=${() => props.onSubsectionChange(ALL_SUBSECTION)}
                >
                  ${t("configSubnavAll")}
                </button>
                ${subsections.map(
                  (entry) => html`
                    <button
                      class="config-subnav__item ${
                        effectiveSubsection === entry.key ? "active" : ""
                      }"
                      title=${entry.description || entry.label}
                      @click=${() => props.onSubsectionChange(entry.key)}
                    >
                      ${entry.label}
                    </button>
                  `,
                )}
              </div>
            `
            : nothing
        }

        <!-- Form content -->
        <div class="config-content ${props.formMode === "raw" ? "config-content--raw" : ""}">
          ${
            props.activeSection === "rbac"
              ? renderRbacManagement(props)
              : props.formMode === "form"
                ? html`
                ${
                  props.schemaLoading
                    ? html`
                        <div class="config-loading">
                          <div class="config-loading__spinner"></div>
                          <span>${t("configLoadingSchema")}</span>
                        </div>
                      `
                    : renderConfigForm({
                        schema: analysis.schema,
                        uiHints: props.uiHints,
                        value: props.formValue,
                        disabled: props.loading || !props.formValue,
                        unsupportedPaths: analysis.unsupportedPaths,
                        onPatch: props.onFormPatch,
                        searchQuery: props.searchQuery,
                        activeSection: props.activeSection,
                        activeSubsection: effectiveSubsection,
                      })
                }
                ${
                  formUnsafe
                    ? html`
                        <div class="callout danger" style="margin-top: 12px">
                          ${t("configFormUnsafeWarning")}
                        </div>
                      `
                    : nothing
                }
              `
              : html`
                <label class="field config-raw-field">
                  <span>${t("configRawJson5")}</span>
                  <span class="textarea"><textarea
                    .value=${props.raw}
                    @input=${(e: Event) =>
                      props.onRawChange((e.target as HTMLTextAreaElement).value)}
                  ></textarea></span>
                </label>
              `
          }
        </div>

        ${
          props.issues.length > 0
            ? html`<div class="callout danger" style="margin-top: 12px;">
              <pre class="code-block">
${JSON.stringify(props.issues, null, 2)}</pre
              >
            </div>`
            : nothing
        }
      </main>
    </div>
  `;
}

function renderRbacManagement(props: ConfigProps) {
  const users = props.rbacUsersList ?? [];
  const roles = props.rbacRolesList ?? [];
  const loading = props.rbacUsersLoading ?? false;

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const usernameInput = form.querySelector("#rbac-username") as HTMLInputElement;
    const passwordInput = form.querySelector("#rbac-password") as HTMLInputElement;
    const roleSelect = form.querySelector("#rbac-role") as HTMLSelectElement;

    const username = usernameInput.value.trim();
    const password = passwordInput.value.trim();
    const roleId = Number(roleSelect.value);

    if (!username || !password || !roleId) {
      alert("请填写完整表单项");
      return;
    }

    if (props.onCreateUser) {
      props.onCreateUser(username, password, roleId)
        .then(() => {
          usernameInput.value = "";
          passwordInput.value = "";
          roleSelect.selectedIndex = 0;
        })
        .catch((err) => {
          alert("创建用户失败: " + err.message);
        });
    }
  };

  return html`
    <div class="env-vars" style="display: flex; flex-direction: column; gap: 24px;">
      <section class="env-vars__section card">
        <h3 class="card-title" style="margin-bottom: 8px;">用户与权限管理</h3>
        <p class="muted" style="font-size: 12px; margin-bottom: 16px;">
          在此管理 OpenOcta 系统的登录用户及其对应的角色权限。默认管理员账号 admin 无法删除。
        </p>

        ${loading
          ? html`<div style="text-align: center; padding: 20px; color: var(--color-text-muted);">正在加载用户列表...</div>`
          : html`
              <table class="env-vars__table">
                <thead>
                  <tr>
                    <th>用户ID</th>
                    <th>用户名</th>
                    <th>角色</th>
                    <th>创建时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  ${users.map(
                    (u: any) => html`
                      <tr>
                        <td style="padding: 12px 8px;">${u.id}</td>
                        <td style="font-weight: 500; color: var(--color-text-accent); padding: 12px 8px;">${u.username}</td>
                        <td style="padding: 12px 8px;">
                          <span class="pill pill--sm" style="background: rgba(255,255,255,0.05); color: #fff;">
                            ${u.roleName}
                          </span>
                        </td>
                        <td style="padding: 12px 8px;">${new Date(u.createdAt).toLocaleString()}</td>
                        <td style="padding: 12px 8px;">
                          ${u.username === "admin"
                            ? html`<span class="muted" style="font-size: 12px;">不可修改</span>`
                            : html`
                                <button
                                  class="btn small btn--ghost"
                                  style="color: var(--color-danger); padding: 2px 6px;"
                                  @click=${() => {
                                    if (confirm(`确认删除用户 ${u.username} 吗？`)) {
                                      props.onDeleteUser?.(u.id);
                                    }
                                  }}
                                >
                                  删除
                                </button>
                              `}
                        </td>
                      </tr>
                    `
                  )}
                </tbody>
              </table>
            `}
      </section>

      <section class="env-vars__section card" style="max-width: 500px;">
        <h3 class="card-title" style="margin-bottom: 12px;">新增系统用户</h3>
        <form @submit=${handleSubmit} style="display: flex; flex-direction: column; gap: 16px;">
          <label class="field">
            <span style="font-size: 13px; font-weight: 500; margin-bottom: 6px;">用户名</span>
            <span class="input">
              <input type="text" id="rbac-username" required placeholder="请输入用户名" />
            </span>
          </label>

          <label class="field">
            <span style="font-size: 13px; font-weight: 500; margin-bottom: 6px;">密码</span>
            <span class="input">
              <input type="password" id="rbac-password" required placeholder="请输入密码" />
            </span>
          </label>

          <label class="field">
            <span style="font-size: 13px; font-weight: 500; margin-bottom: 6px;">角色</span>
            <span class="input" style="background: var(--color-bg-content); border-radius: 4px; border: 1px solid var(--color-border);">
              <select id="rbac-role" style="background: transparent; border: none; color: #fff; width: 100%; height: 36px; padding: 0 8px; outline: none; cursor: pointer;">
                ${roles.map((r: any) => html`<option value=${r.id} style="background: var(--color-bg-sidebar); color: #fff;">${r.description || r.name}</option>`)}
              </select>
            </span>
          </label>

          <div style="margin-top: 8px;">
            <button type="submit" class="btn btn--sm primary">确认添加</button>
          </div>
        </form>
      </section>
    </div>
  `;
}
