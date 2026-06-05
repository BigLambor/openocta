import { LitElement, css, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import {
  DOMAIN_MONITOR_GUIDES,
  validateMonitorLabelsForCluster,
  type DomainMonitorGuide,
} from "../utils/monitor-labels.ts";

@customElement("ops-monitor-labels-field")
export class OpsMonitorLabelsField extends LitElement {
  @property({ type: String }) domain = "hadoop";
  @property({ type: String }) status = "unknown";

  @state() private value = "";
  @state() private touched = false;
  @state() private error: string | null = null;

  get inputValue(): string {
    const result = validateMonitorLabelsForCluster(this.domain, this.status, this.value);
    return result.ok ? (result.normalized ?? "") : this.value.trim();
  }

  focus() {
    this.renderRoot.querySelector("input")?.focus();
  }

  checkValidity(): boolean {
    this.touched = true;
    return this.runValidation();
  }

  private runValidation(): boolean {
    const result = validateMonitorLabelsForCluster(this.domain, this.status, this.value);
    this.error = result.ok ? null : (result.error ?? "校验失败");
    return result.ok;
  }

  private guide(): DomainMonitorGuide | undefined {
    return DOMAIN_MONITOR_GUIDES[this.domain];
  }

  private onInput(e: Event) {
    this.value = (e.target as HTMLInputElement).value;
    if (this.touched) this.runValidation();
  }

  private onBlur() {
    this.touched = true;
    this.runValidation();
  }

  render() {
    const guide = this.guide();
    return html`
      <div class="wrap">
        <div class="chain">
          <span class="chain__step">资产 id (cluster-uuid)</span>
          <span class="chain__arrow">→</span>
          <span class="chain__step chain__step--emph">monitorLabels</span>
          <span class="chain__arrow">→</span>
          <span class="chain__step">InjectLabelsIntoPromQL</span>
          <span class="chain__arrow">→</span>
          <span class="chain__step">VM 时序标签</span>
        </div>

        <label class="field">
          <span>监控标签 (monitorLabels) <strong class="req">*</strong></span>
          <input
            class="input ${this.error && this.touched ? "input--error" : ""}"
            .value=${this.value}
            placeholder=${guide?.example ?? 'job="prod",cluster="a"'}
            @input=${this.onInput}
            @blur=${this.onBlur}
          />
        </label>

        ${this.error && this.touched
          ? html`<p class="msg msg--error">${this.error}</p>`
          : nothing}

        ${guide
          ? html`
              <div class="hint">
                <p class="hint__title">${guide.domainLabel} 对齐要求</p>
                <p>须包含标签之一：<code>${guide.labelKeys.join(", ")}</code></p>
                <p>示例：<code>${guide.example}</code></p>
                <p class="hint__muted">
                  域级探测基线：<code>${guide.baseQueryHint}</code>；登记后系统会把你的
                  monitorLabels 注入到该查询中。
                </p>
                <details class="hint__details">
                  <summary>验收步骤</summary>
                  <ol>
                    ${guide.checkSteps.map((s) => html`<li>${s}</li>`)}
                    <li>
                      VM 验证：将登记的 monitorLabels 并入选择器，例如
                      <code>count(up{${guide.example.split(",")[0]},...})</code>
                    </li>
                  </ol>
                </details>
              </div>
            `
          : nothing}
      </div>
    `;
  }

  static styles = css`
    .wrap {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .chain {
      display: flex;
      flex-wrap: wrap;
      gap: 4px 6px;
      align-items: center;
      font-size: 11px;
      color: var(--text-muted, #94a3b8);
      padding: 8px 10px;
      border-radius: 8px;
      border: 1px dashed var(--border, #334155);
      background: rgba(148, 163, 184, 0.06);
    }
    .chain__step--emph {
      color: var(--text-primary, #e2e8f0);
      font-weight: 600;
    }
    .chain__arrow {
      opacity: 0.7;
    }
    .field {
      display: flex;
      flex-direction: column;
      gap: 6px;
      font-size: 12px;
      color: var(--text-secondary, #cbd5e1);
    }
    .req {
      color: #f59e0b;
      font-weight: 600;
    }
    .input {
      padding: 8px 10px;
      border-radius: 8px;
      border: 1px solid var(--border, #334155);
      background: var(--bg, #0f172a);
      color: var(--text-primary, #e2e8f0);
      font-size: 13px;
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    }
    .input--error {
      border-color: #ef4444;
    }
    .msg {
      margin: 0;
      font-size: 12px;
    }
    .msg--error {
      color: #ef4444;
    }
    .hint {
      font-size: 12px;
      color: var(--text-secondary, #cbd5e1);
      padding: 10px 12px;
      border-radius: 8px;
      background: var(--bg-content, rgba(15, 23, 42, 0.5));
      border: 1px solid var(--border, #334155);
    }
    .hint__title {
      margin: 0 0 6px;
      font-weight: 600;
      color: var(--text-primary, #e2e8f0);
    }
    .hint p {
      margin: 4px 0;
    }
    .hint code {
      font-size: 11px;
      word-break: break-all;
    }
    .hint__muted {
      color: var(--text-muted, #94a3b8);
    }
    .hint__details {
      margin-top: 8px;
    }
    .hint__details summary {
      cursor: pointer;
      color: var(--text-primary, #e2e8f0);
    }
    .hint__details ol {
      margin: 8px 0 0;
      padding-left: 18px;
    }
  `;
}

declare global {
  interface HTMLElementTagNameMap {
    "ops-monitor-labels-field": OpsMonitorLabelsField;
  }
}
