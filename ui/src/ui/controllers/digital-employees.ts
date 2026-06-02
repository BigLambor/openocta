import type { AppViewState } from "../app-view-state.ts";
import { gatewayHttpBase } from "../gateway-url.ts";

type EmployeesListResult = {
  employees: Array<{
    id: string;
    name: string;
    description: string;
    builtin: boolean;
    from?: string;
    type?: string;
    prompt?: string;
    enabled?: boolean;
    createdAt?: number;
    skillIds?: string[];
    skillNames?: string[];
    mcpServerKeys?: string[];
    domainKeys?: string[];
    capabilityKeys?: string[];
    roleType?: string;
    responsibilities?: string[];
    inputSources?: string[];
    outputTypes?: string[];
    actionScopes?: string[];
    permissionKeys?: string[];
    runbookRefs?: string[];
    knowledgeRefs?: string[];
  }>;
};

export async function loadDigitalEmployees(state: AppViewState) {
  if (!state.client || !state.connected) {
    return;
  }
  state.digitalEmployeesLoading = true;
  state.digitalEmployeesError = null;
  try {
    const res = await state.client.request<EmployeesListResult>("employees.list", {});
    state.digitalEmployees = res?.employees ?? [];
  } catch (err) {
    state.digitalEmployeesError = String(err);
  } finally {
    state.digitalEmployeesLoading = false;
  }
}

/** 从文件名推导技能名称，如 prometheus-1.0.0.zip -> prometheus-1.0.0 */
function deriveSkillNameFromFileName(fileName: string): string {
  let base = fileName.trim();
  if (!base) return "";
  // 移除常见扩展名（不区分大小写）
  const extPatterns = [/\.zip$/i, /\.tar\.gz$/i, /\.tgz$/i, /\.md$/i];
  for (const re of extPatterns) {
    base = base.replace(re, "");
  }
  return base.trim() || "";
}

export async function createDigitalEmployee(state: AppViewState) {
  if (!state.client || !state.connected) {
    return;
  }
  const name = state.digitalEmployeeCreateName?.trim();
  if (!name) {
    state.digitalEmployeeCreateError = "名称不能为空";
    return;
  }
  state.digitalEmployeeCreateBusy = true;
  state.digitalEmployeeCreateError = null;
  state.digitalEmployeeSkillUploadError = null;
  let mcpServers: Record<string, unknown> | undefined;
  const mcpJson = state.digitalEmployeeCreateMcpJson?.trim();
  if (mcpJson) {
    try {
      const parsed = JSON.parse(mcpJson) as Record<string, unknown>;
      if (parsed && typeof parsed === "object" && Object.keys(parsed).length > 0) {
        mcpServers = parsed;
      }
    } catch {
      state.digitalEmployeeCreateError = "MCP 配置 JSON 格式无效";
      state.digitalEmployeeCreateBusy = false;
      return;
    }
  }
  try {
    const payload: Record<string, unknown> = {
      name,
      description: state.digitalEmployeeCreateDescription ?? "",
      prompt: state.digitalEmployeeCreatePrompt ?? "",
      enabled: true,
      domainKeys: state.digitalEmployeeCreateDomainKeys ?? [],
      capabilityKeys: state.digitalEmployeeCreateCapabilityKeys ?? [],
      roleType: state.digitalEmployeeCreateRoleType ?? "",
      responsibilities: state.digitalEmployeeCreateResponsibilities ?? [],
      inputSources: state.digitalEmployeeCreateInputSources ?? [],
      outputTypes: state.digitalEmployeeCreateOutputTypes ?? [],
      actionScopes: state.digitalEmployeeCreateActionScopes ?? [],
      permissionKeys: state.digitalEmployeeCreatePermissionKeys ?? [],
      runbookRefs: state.digitalEmployeeCreateRunbookRefs ?? [],
      knowledgeRefs: state.digitalEmployeeCreateKnowledgeRefs ?? [],
    };
    if (mcpServers) payload.mcpServers = mcpServers;
    const res = await state.client.request<{ id: string }>("employees.create", payload);
    const employeeId = res?.id ?? deriveEmployeeIdFromName(name);

    // 若有技能文件，一并上传（提交时一次完成，支持多文件）
    const files = state.digitalEmployeeSkillUploadFiles ?? [];
    const baseSkillName = state.digitalEmployeeSkillUploadName?.trim();
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const skillName =
        baseSkillName && files.length === 1
          ? baseSkillName
          : deriveSkillNameFromFileName(file.name);
      const uploadResult = await uploadEmployeeSkill(state, employeeId, skillName, file);
      if (!uploadResult.ok) {
        state.digitalEmployeeCreateError = uploadResult.error ?? "技能文件上传失败";
        return;
      }
    }

    state.digitalEmployeeCreateName = "";
    state.digitalEmployeeCreateDescription = "";
    state.digitalEmployeeCreatePrompt = "";
    state.digitalEmployeeCreateMcpJson = "";
    state.digitalEmployeeSkillUploadName = "";
    state.digitalEmployeeSkillUploadFiles = [];
    state.digitalEmployeeSkillUploadError = null;
    state.digitalEmployeeCreateDomainKeys = [];
    state.digitalEmployeeCreateCapabilityKeys = [];
    state.digitalEmployeeCreateRoleType = "";
    state.digitalEmployeeCreateResponsibilities = [];
    state.digitalEmployeeCreateInputSources = [];
    state.digitalEmployeeCreateOutputTypes = [];
    state.digitalEmployeeCreateActionScopes = [];
    state.digitalEmployeeCreatePermissionKeys = [];
    state.digitalEmployeeCreateRunbookRefs = [];
    state.digitalEmployeeCreateKnowledgeRefs = [];
    await loadDigitalEmployees(state);
  } catch (err) {
    state.digitalEmployeeCreateError = String(err);
  } finally {
    state.digitalEmployeeCreateBusy = false;
  }
}

function deriveEmployeeIdFromName(name: string): string {
  const s = name.trim().toLowerCase();
  if (!s) return "employee";
  let out = "";
  for (const ch of s) {
    if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9")) out += ch;
    else if (ch === "-" || ch === "_" || ch === " ") out += "-";
  }
  out = out.replace(/-+/g, "-").replace(/^-+/, "").replace(/-+$/, "");
  return out || "employee";
}

export async function updateDigitalEmployeeEnabled(
  state: AppViewState,
  id: string,
  enabled: boolean,
) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    await state.client.request("employees.create", { id, enabled });
    await loadDigitalEmployees(state);
  } catch (err) {
    state.digitalEmployeesError = String(err);
  }
}

export async function deleteDigitalEmployee(state: AppViewState, id: string) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    await state.client.request("employees.delete", { id });
    await loadDigitalEmployees(state);
  } catch (err) {
    state.digitalEmployeesError = String(err);
  }
}

type EmployeeManifest = {
  id: string;
  name: string;
  description: string;
  prompt?: string;
  enabled?: boolean;
  skillIds?: string[];
  mcpServers?: Record<string, unknown>;
  domainKeys?: string[];
  capabilityKeys?: string[];
  roleType?: string;
  responsibilities?: string[];
  inputSources?: string[];
  outputTypes?: string[];
  actionScopes?: string[];
  permissionKeys?: string[];
  runbookRefs?: string[];
  knowledgeRefs?: string[];
};

export async function getDigitalEmployee(
  state: AppViewState,
  id: string,
): Promise<EmployeeManifest | null> {
  if (!state.client || !state.connected) {
    return null;
  }
  try {
    const res = await state.client.request<EmployeeManifest>("employees.get", { id });
    return res ?? null;
  } catch {
    return null;
  }
}

function normalizeNameForCompare(name: string): string {
  return name.trim().toLowerCase();
}

function deriveCopyName(baseName: string, existingNames: string[]): string {
  const base = baseName.trim() || "employee";
  const normalizedExisting = new Set(existingNames.map(normalizeNameForCompare));
  const candidate1 = `${base} copy`;
  if (!normalizedExisting.has(normalizeNameForCompare(candidate1))) {
    return candidate1;
  }
  for (let i = 2; i <= 99; i++) {
    const candidate = `${base} copy ${i}`;
    if (!normalizedExisting.has(normalizeNameForCompare(candidate))) {
      return candidate;
    }
  }
  return `${base} copy ${Date.now()}`;
}

export async function copyDigitalEmployee(state: AppViewState, id: string) {
  if (!state.client || !state.connected) {
    return;
  }
  state.digitalEmployeesError = null;
  state.digitalEmployeesLoading = true;
  try {
    const manifest = await getDigitalEmployee(state, id);
    if (!manifest) {
      state.digitalEmployeesError = "无法加载员工详情";
      return;
    }
    const baseName = (manifest.name || manifest.id || id).trim();
    const nextName = deriveCopyName(
      baseName || "employee",
      (state.digitalEmployees ?? []).map((e) => e.name || ""),
    );
    const payload: Record<string, unknown> = {
      name: nextName,
      description: manifest.description ?? "",
      prompt: manifest.prompt ?? "",
      enabled: manifest.enabled !== false,
    };
    if (manifest.mcpServers) {
      payload.mcpServers = manifest.mcpServers;
    }
    if (Array.isArray(manifest.skillIds) && manifest.skillIds.length > 0) {
      payload.skillIds = manifest.skillIds;
    }
    for (const key of [
      "domainKeys",
      "capabilityKeys",
      "responsibilities",
      "inputSources",
      "outputTypes",
      "actionScopes",
      "permissionKeys",
      "runbookRefs",
      "knowledgeRefs",
    ] as const) {
      if (Array.isArray(manifest[key]) && manifest[key]!.length > 0) {
        payload[key] = manifest[key];
      }
    }
    if (manifest.roleType) {
      payload.roleType = manifest.roleType;
    }
    await state.client.request<{ id: string }>("employees.create", payload);
    await loadDigitalEmployees(state);
  } catch (err) {
    state.digitalEmployeesError = String(err);
  } finally {
    state.digitalEmployeesLoading = false;
  }
}

export async function updateDigitalEmployee(state: AppViewState) {
  if (!state.client || !state.connected) {
    return;
  }
  const id = state.digitalEmployeeEditId?.trim();
  if (!id) {
    state.digitalEmployeeEditError = "员工 ID 不能为空";
    return;
  }
  state.digitalEmployeeEditBusy = true;
  state.digitalEmployeeEditError = null;
  let mcpServers: Record<string, unknown> | undefined;
  const mcpJson = state.digitalEmployeeEditMcpJson?.trim();
  try {
    if (mcpJson) {
      const parsed = JSON.parse(mcpJson) as Record<string, unknown>;
      if (parsed && typeof parsed === "object") {
        mcpServers = parsed;
      }
    } else {
      mcpServers = {};
    }
  } catch {
    state.digitalEmployeeEditError = "MCP 配置 JSON 格式无效";
    state.digitalEmployeeEditBusy = false;
    return;
  }
  try {
    const payload: Record<string, unknown> = {
      id,
      description: state.digitalEmployeeEditDescription ?? "",
      prompt: state.digitalEmployeeEditPrompt ?? "",
      enabled: state.digitalEmployeeEditEnabled !== false,
      mcpServers: mcpServers ?? {},
      domainKeys: state.digitalEmployeeEditDomainKeys ?? [],
      capabilityKeys: state.digitalEmployeeEditCapabilityKeys ?? [],
      roleType: state.digitalEmployeeEditRoleType ?? "",
      responsibilities: state.digitalEmployeeEditResponsibilities ?? [],
      inputSources: state.digitalEmployeeEditInputSources ?? [],
      outputTypes: state.digitalEmployeeEditOutputTypes ?? [],
      actionScopes: state.digitalEmployeeEditActionScopes ?? [],
      permissionKeys: state.digitalEmployeeEditPermissionKeys ?? [],
      runbookRefs: state.digitalEmployeeEditRunbookRefs ?? [],
      knowledgeRefs: state.digitalEmployeeEditKnowledgeRefs ?? [],
    };
    await state.client.request("employees.create", payload);

    // 删除标记的技能
    for (const name of state.digitalEmployeeEditSkillsToDelete ?? []) {
      const ok = await deleteEmployeeSkill(state, id, name);
      if (!ok) {
        state.digitalEmployeeEditError = `删除技能 ${name} 失败`;
        return;
      }
    }

    // 上传新技能文件
    const files = state.digitalEmployeeEditSkillFilesToUpload ?? [];
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const skillName = deriveSkillNameFromFileName(file.name);
      const uploadResult = await uploadEmployeeSkill(state, id, skillName, file);
      if (!uploadResult.ok) {
        state.digitalEmployeeEditError = uploadResult.error ?? "技能文件上传失败";
        return;
      }
    }

    state.digitalEmployeeEditModalOpen = false;
    state.digitalEmployeeEditId = "";
    state.digitalEmployeeEditName = "";
    state.digitalEmployeeEditDescription = "";
    state.digitalEmployeeEditPrompt = "";
    state.digitalEmployeeEditMcpJson = "";
    state.digitalEmployeeEditSkillNames = [];
    state.digitalEmployeeEditSkillFilesToUpload = [];
    state.digitalEmployeeEditSkillsToDelete = [];
    state.digitalEmployeeEditDomainKeys = [];
    state.digitalEmployeeEditCapabilityKeys = [];
    state.digitalEmployeeEditRoleType = "";
    state.digitalEmployeeEditResponsibilities = [];
    state.digitalEmployeeEditInputSources = [];
    state.digitalEmployeeEditOutputTypes = [];
    state.digitalEmployeeEditActionScopes = [];
    state.digitalEmployeeEditPermissionKeys = [];
    state.digitalEmployeeEditRunbookRefs = [];
    state.digitalEmployeeEditKnowledgeRefs = [];
    await loadDigitalEmployees(state);
  } catch (err) {
    state.digitalEmployeeEditError = String(err);
  } finally {
    state.digitalEmployeeEditBusy = false;
  }
}

export async function deleteEmployeeSkill(
  state: AppViewState,
  employeeId: string,
  name: string,
): Promise<boolean> {
  const gatewayUrl = state.settings.gatewayUrl?.trim();
  if (!gatewayUrl) {
    return false;
  }
  const url = gatewayHttpBase(gatewayUrl);
  if (!url) {
    return false;
  }
  const headers: Record<string, string> = {};
  const token = (state as { settings?: { token?: string } }).settings?.token?.trim();
  if (token) headers["Authorization"] = `Bearer ${token}`;
  try {
    const u = new URL(`${url.replace(/\/$/, "")}/api/employee-skills/delete`);
    u.searchParams.set("employeeId", employeeId.trim());
    u.searchParams.set("name", name.trim());
    const res = await fetch(u.toString(), { method: "DELETE", headers });
    if (res.status === 401) {
      throw new Error("认证失败：网关令牌无效或未提供，请在 Overview 中配置正确的 Gateway Token");
    }
    const data = (await res.json()) as { ok?: boolean };
    return res.ok && data.ok === true;
  } catch (err) {
    const raw = err instanceof Error ? err.message : String(err);
    if (raw === "Failed to fetch") {
      throw new Error("网络请求失败，请检查网络连接");
    }
    throw err;
  }
}

export type EmployeeSkillsUploadResult = {
  ok: boolean;
  error?: string;
  template?: string;
};

export async function uploadEmployeeSkill(
  state: AppViewState,
  employeeId: string,
  name: string,
  file: File,
): Promise<EmployeeSkillsUploadResult> {
  const gatewayUrl = state.settings.gatewayUrl?.trim();
  if (!gatewayUrl) {
    return { ok: false, error: "Gateway URL 未配置" };
  }
  const url = gatewayHttpBase(gatewayUrl);
  if (!url) {
    return { ok: false, error: "Gateway URL 无效" };
  }
  const form = new FormData();
  form.append("employeeId", employeeId.trim());
  if (name.trim()) {
    form.append("name", name.trim());
  }
  form.append("file", file);
  const headers: Record<string, string> = {};
  const token = (state as { settings?: { token?: string } }).settings?.token?.trim();
  if (token) headers["Authorization"] = `Bearer ${token}`;
  try {
    const res = await fetch(`${url.replace(/\/$/, "")}/api/employee-skills/upload`, {
      method: "POST",
      headers,
      body: form,
    });
    const data = (await res.json()) as { ok?: boolean; error?: string; template?: string };
    if (!res.ok || data.ok === false) {
      const errMsg =
        res.status === 401
          ? "认证失败：网关令牌无效或未提供，请在 Overview 中配置正确的 Gateway Token"
          : (data.error ?? `上传失败 (${res.status})`);
      return { ok: false, error: errMsg, template: data.template };
    }
    return { ok: true };
  } catch (err) {
    const raw = err instanceof Error ? err.message : String(err);
    const msg = raw === "Failed to fetch" ? "网络请求失败，请检查网络连接" : raw;
    return { ok: false, error: msg };
  }
}
