import { ReactiveController, ReactiveControllerHost } from "lit";
import type { ConfigSnapshot, ConfigUiHints } from "../types.ts";

export class ConfigStore implements ReactiveController {
  host: ReactiveControllerHost;

  configLoading = false;
  configRaw = "{\n}\n";
  configRawOriginal = "";
  configValid: boolean | null = null;
  configIssues: unknown[] = [];
  configSaving = false;
  configApplying = false;
  configSnapshot: ConfigSnapshot | null = null;
  configSchema: unknown = null;
  configSchemaVersion: string | null = null;
  configSchemaLoading = false;
  configUiHints: ConfigUiHints = {};
  configForm: Record<string, unknown> | null = null;
  configFormOriginal: Record<string, unknown> | null = null;
  configFormDirty = false;
  configFormMode: "form" | "raw" = "raw";
  configSearchQuery = "";
  configActiveSection: string | null = null;
  configActiveSubsection: string | null = null;

  constructor(host: ReactiveControllerHost) {
    this.host = host;
    host.addController(this);
  }

  hostConnected() {}
  hostDisconnected() {}
}
