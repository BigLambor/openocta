/** Parse inspection cron output into executive summary vs full report body. */

export type InspectionResultLike = {
  score?: number | null;
  scoreStatus?: string;
  errors?: string[];
  metricsEvidence?: Record<string, unknown>;
  reportMarkdown?: string;
};

export type InspectionRunLike = {
  summary?: string;
  error?: string;
  status?: string;
  result?: InspectionResultLike;
};

const JSON_FENCE_RE = /```(?:json)?\s*[\s\S]*?```/gi;
const SUMMARY_HEADING_RE =
  /^##\s*(?:[一二三四五六七八九十\d]+、\s*)?(?:执行摘要|发现摘要|巡检总结|巡检发现摘要)\s*$/im;
const RISK_HEADING_RE = /^###\s*P\d+\s*[·•\-]/im;

function stripJsonFences(text: string): string {
  return text.replace(JSON_FENCE_RE, "").trim();
}

function stripLeadingJsonObject(text: string): string {
  const trimmed = text.trim();
  if (!trimmed.startsWith("{")) {
    return trimmed;
  }
  const end = trimmed.lastIndexOf("}");
  if (end <= 0) {
    return trimmed;
  }
  const rest = trimmed.slice(end + 1).trim();
  return rest || trimmed;
}

/** Best-effort pick of the richest report source on a cron run entry. */
export function pickInspectionReportSource(entry: InspectionRunLike): string {
  const resultMd = String(entry.result?.reportMarkdown ?? "").trim();
  const summary = String(entry.summary ?? "").trim();
  if (summary && (summary.includes("##") || summary.length > resultMd.length + 80)) {
    return summary;
  }
  if (resultMd && !resultMd.startsWith("{")) {
    return resultMd;
  }
  return resultMd || summary;
}

/** Full markdown body for the detail panel (no JSON payload noise). */
export function normalizeInspectionReportMarkdown(raw: string): string {
  const text = stripLeadingJsonObject(stripJsonFences(raw)).trim();
  return text;
}

function extractSummarySectionBody(body: string): string | null {
  const match = body.match(SUMMARY_HEADING_RE);
  if (!match || match.index == null) {
    return null;
  }
  const start = match.index + match[0].length;
  const rest = body.slice(start);
  const nextHeading = rest.search(/^##\s+/m);
  return (nextHeading >= 0 ? rest.slice(0, nextHeading) : rest).trim() || null;
}

/** Lead paragraphs only — stops before tables, lists, or horizontal rules. */
function extractLeadParagraphs(section: string): string {
  const paragraphs: string[] = [];
  let current: string[] = [];

  for (const line of section.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) {
      if (current.length) {
        paragraphs.push(current.join(" "));
        current = [];
      }
      continue;
    }
    if (
      trimmed.startsWith("|") ||
      trimmed.startsWith("---") ||
      trimmed.startsWith("#") ||
      /^[-*+]\s+/.test(trimmed) ||
      /^\d+\.\s+/.test(trimmed)
    ) {
      break;
    }
    current.push(trimmed);
  }
  if (current.length) {
    paragraphs.push(current.join(" "));
  }

  return paragraphs
    .map((p) =>
      p
        .replace(/!\[[^\]]*]\([^)]*\)/g, "")
        .replace(/\[([^\]]+)]\([^)]*\)/g, "$1")
        .replace(/`{1,3}([^`]+)`{1,3}/g, "$1")
        .replace(/\*\*([^*]+)\*\*/g, "$1")
        .replace(/\*([^*]+)\*/g, "$1")
        .trim(),
    )
    .filter(Boolean)
    .join("\n\n");
}

/** Extract the executive-summary lead text; returns null when not found. */
export function extractInspectionExecutiveSummary(raw: string): string | null {
  const body = normalizeInspectionReportMarkdown(raw);
  if (!body) {
    return null;
  }
  const section = extractSummarySectionBody(body);
  if (!section) {
    return null;
  }
  const lead = extractLeadParagraphs(section);
  return lead || null;
}

function scoreStatusLabel(score: number | null, scoreStatus?: string): string {
  if (scoreStatus === "ok" || (score != null && score >= 90)) {
    return "健康";
  }
  if (scoreStatus === "warning" || (score != null && score >= 75)) {
    return "亚健康";
  }
  if (score != null && score >= 0) {
    return "风险";
  }
  return "未知";
}

function extractRiskHighlights(markdown: string, limit = 3): string[] {
  const body = normalizeInspectionReportMarkdown(markdown);
  if (!body) {
    return [];
  }
  const risks: string[] = [];
  const lines = body.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (RISK_HEADING_RE.test(trimmed)) {
      risks.push(trimmed.replace(/^###\s*/, "").replace(/\s+/g, " "));
    }
  }
  return risks.slice(0, limit);
}

export function buildInspectionFallbackSummary(
  entry: InspectionRunLike,
  score: number | null,
): string {
  if (entry.error || entry.status === "error") {
    return `巡检执行失败：${entry.error || "未知错误"}`;
  }

  const errors = entry.result?.errors?.filter(Boolean) ?? [];
  if (errors.length > 0) {
    const head = errors.slice(0, 2).join("；");
    return errors.length > 2 ? `${head} 等 ${errors.length} 项异常` : head;
  }

  const status = entry.result?.scoreStatus ?? "";
  if (score != null && Number.isFinite(score)) {
    const label = scoreStatusLabel(score, status);
    return `综合健康分 ${score}/100（${label}），详见完整报告中的风险项与处置建议。`;
  }

  return "巡检已完成，请查看完整报告。";
}

/** Structured bullets for the detail panel — not a copy of the full report. */
export function buildInspectionDetailBullets(
  entry: InspectionRunLike,
  score: number | null,
  reportMarkdown: string,
): string[] {
  const bullets: string[] = [];
  if (score != null && Number.isFinite(score)) {
    bullets.push(
      `综合健康分 ${score}/100（${scoreStatusLabel(score, entry.result?.scoreStatus)}）`,
    );
  }

  const conclusion = extractInspectionExecutiveSummary(reportMarkdown);
  if (conclusion) {
    bullets.push(conclusion);
  }

  const risks = extractRiskHighlights(reportMarkdown);
  if (risks.length > 0) {
    bullets.push(`优先处置：${risks.join("；")}`);
  } else if (entry.result?.errors?.length) {
    bullets.push(`异常项：${entry.result.errors.slice(0, 2).join("；")}`);
  }

  if (bullets.length === 0) {
    return [buildInspectionFallbackSummary(entry, score)];
  }
  return bullets;
}

/** One-line preview for the report list. */
export function buildInspectionListPreview(summary: string, maxLen = 80): string {
  const firstLine = summary
    .split("\n")
    .map((line) => line.trim())
    .find(Boolean) ?? summary;
  const flat = firstLine.replace(/\s+/g, " ").trim();
  if (flat.length <= maxLen) {
    return flat;
  }
  return `${flat.slice(0, Math.max(0, maxLen - 1))}…`;
}

export function resolveInspectionSummaries(
  entry: InspectionRunLike,
  score: number | null,
): { reportSummary: string; reportMarkdown: string; reportSummaryBullets: string[] } {
  const raw = pickInspectionReportSource(entry);
  const reportMarkdown =
    normalizeInspectionReportMarkdown(raw) ||
    (entry.error ? `### 巡检失败\n- **原因**：${entry.error}` : "");

  const reportSummaryBullets = buildInspectionDetailBullets(entry, score, reportMarkdown);
  const reportSummary = reportSummaryBullets.join("\n");

  return { reportSummary, reportMarkdown, reportSummaryBullets };
}
