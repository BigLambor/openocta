import type { AppViewState } from "../app-view-state.ts";

export type EmployeeTask = {
  id: string;
  employeeId: string;
  domainKey: string;
  capabilityKey: string;
  scenarioKey: string;
  objectRef: string;
  triggerType: string;
  status: string;
  input: string;
  output: string;
  conclusion: string;
  artifacts?: string[];
  startedAt: number;
  finishedAt: number;
  operator: string;
  evaluation: string;
};

export type EmployeeEffectiveness = {
  taskCount: number;
  autoCloseRate: number;
  adoptionRate: number;
  noiseReductionRate: number;
  savedHours: number;
  costSpent: number;
  taskBreakdown: Record<string, number>;
  domainBreakdown: Record<string, number>;
};

export async function loadEmployeeTasks(state: AppViewState) {
  if (!state.client || !state.connected) {
    return;
  }
  state.employeeTasksLoading = true;
  state.employeeTasksError = null;
  try {
    const params: Record<string, unknown> = {};
    if (state.employeeTaskFilterEmployee) {
      params["employeeId"] = state.employeeTaskFilterEmployee;
    }
    if (state.employeeTaskFilterStatus) {
      params["status"] = state.employeeTaskFilterStatus;
    }
    if (state.employeeTaskFilterQuery) {
      params["query"] = state.employeeTaskFilterQuery;
    }
    const res = await state.client.request<{ tasks: EmployeeTask[] }>("employee.tasks.list", params);
    state.employeeTasks = res?.tasks ?? [];
  } catch (err) {
    state.employeeTasksError = String(err);
  } finally {
    state.employeeTasksLoading = false;
  }
}

export async function loadEmployeeEffectiveness(state: AppViewState) {
  if (!state.client || !state.connected) {
    return;
  }
  state.employeeEffectivenessLoading = true;
  state.employeeEffectivenessError = null;
  try {
    const res = await state.client.request<EmployeeEffectiveness>("employee.effectiveness.get", {});
    state.employeeEffectiveness = res ?? null;
  } catch (err) {
    state.employeeEffectivenessError = String(err);
  } finally {
    state.employeeEffectivenessLoading = false;
  }
}

export async function rateEmployeeTask(
  state: AppViewState,
  id: string,
  evaluation: "accepted" | "rejected",
) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    await state.client.request("employee.tasks.update", { id, evaluation });
    // Reload local list and metrics
    void loadEmployeeTasks(state);
    void loadEmployeeEffectiveness(state);
    // Sync task details if active
    if (state.employeeTaskActive && state.employeeTaskActive.id === id) {
      state.employeeTaskActive = { ...state.employeeTaskActive, evaluation };
    }
  } catch (err) {
    state.employeeTasksError = "更新评价失败: " + String(err);
  }
}

export async function deleteEmployeeTask(state: AppViewState, id: string) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    await state.client.request("employee.tasks.delete", { id });
    if (state.employeeTaskActive && state.employeeTaskActive.id === id) {
      state.employeeTaskActive = null;
    }
    void loadEmployeeTasks(state);
    void loadEmployeeEffectiveness(state);
  } catch (err) {
    state.employeeTasksError = "删除记录失败: " + String(err);
  }
}
