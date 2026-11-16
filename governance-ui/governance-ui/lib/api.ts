// API client — all requests go through this module.
// Base URL is read from NEXT_PUBLIC_API_URL env (defaults to localhost:8080 for dev).

const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function getToken(): string {
  if (typeof window === "undefined") return process.env.NEXT_PUBLIC_DEV_API_KEY ?? "";
  // Prefer a token the user stored after login; fall back to the dev env var.
  return localStorage.getItem("cx_token") ?? process.env.NEXT_PUBLIC_DEV_API_KEY ?? "";
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`${res.status}: ${err}`);
  }
  // HEAD responses have no body
  if (init?.method === "HEAD") return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  // Overview
  overview: () => request<Overview>("/api/v1/overview"),

  // Policies
  policies:       (params?: Record<string, string>) => request<PolicyListItem[]>(`/api/v1/policies${toQuery(params)}`),
  createPolicy:   (body: CreatePolicyBody) => request<{ policyId: string; policyVersionId: string }>("/api/v1/policies", { method: "POST", body: JSON.stringify(body) }),
  updatePolicy:   (id: string, body: UpdatePolicyBody) => request<{ policyId: string; packVersion: number }>(`/api/v1/policies/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  activatePolicy: (id: string, body: { policyVersionId: string }) => request<ActivateResponse>(`/api/v1/policies/${id}/activate`, { method: "PUT", body: JSON.stringify(body) }),
  disablePolicy:  (id: string) => request<{ status: string }>(`/api/v1/policies/${id}/disable`, { method: "PUT" }),
  enablePolicy:   (id: string) => request<{ status: string }>(`/api/v1/policies/${id}/enable`, { method: "PUT" }),
  deletePolicy:   (id: string) => request<{ deleted: boolean }>(`/api/v1/policies/${id}`, { method: "DELETE" }),
  policyHistory:  (id: string) => request<PolicyHistoryEntry[]>(`/api/v1/policies/${id}/history`),

  // Sessions
  sessions:        (params?: Record<string, string>) => request<Session[]>(`/api/v1/sessions${toQuery(params)}`),
  sessionTimeline: (id: string) => request<PolicyAction[]>(`/api/v1/sessions/${id}/timeline`),
  sessionHooks:    (id: string) => request<SessionHook[]>(`/api/v1/sessions/${id}/hooks`),
  sessionUsage:    (id: string) => request<SessionUsageDetail>(`/api/v1/sessions/${id}/usage`),

  // Token usage
  tokenUsage:      (days = 30) => request<TokenUsageAnalytics>(`/api/v1/analytics/token-usage?days=${days}`),
  modelPrices:     () => request<ModelPrice[]>("/api/v1/token-usage/prices"),
  exportSession:   (id: string, format: "csv" | "xlsx") => downloadFile(`/api/v1/sessions/${id}/timeline/export?format=${format}`, `session-${id}.${format}`),

  // Violations
  violations:       (params?: Record<string, string>) => request<PolicyAction[]>(`/api/v1/violations${toQuery(params)}`),
  exportViolations: (params: Record<string, string>, format: "csv" | "xlsx") => downloadFile(`/api/v1/violations/export?format=${format}&${new URLSearchParams(params)}`, `violations.${format}`),

  // Developers
  developers: (params?: Record<string, string>) => request<Developer[]>(`/api/v1/developers/list${toQuery(params)}`),

  // Agents
  agents: () => request<Agent[]>("/api/v1/agents"),

  // MCP Servers
  mcpServers:  () => request<MCPServer[]>("/api/v1/mcp-servers"),
  mcpServer:   (id: string) => request<MCPServer>(`/api/v1/mcp-servers/${id}`),

  // Skills
  skills:        () => request<Skill[]>("/api/v1/skills"),
  skillDetail:   (id: string) => request<SkillDetail>(`/api/v1/skills/${id}`),
  sessionSkills: (sessionId: string) => request<SessionSkill[]>(`/api/v1/sessions/${sessionId}/skills`),

  // Compliance
  complianceFrameworks: () => request<ComplianceFramework[]>("/api/v1/compliance/frameworks"),
  frameworkControls:    (id: string) => request<FrameworkControl[]>(`/api/v1/compliance/frameworks/${id}/controls`),
  frameworkStatus:      (id: string) => request<ComplianceFramework>(`/api/v1/compliance/frameworks/${id}/status`),
  controlEvidence:      (id: string) => request<ControlEvidence[]>(`/api/v1/compliance/controls/${id}/evidence`),

  // Audit Log
  auditLog: (params?: Record<string, string>) => request<AuditEvent[]>(`/api/v1/audit${toQuery(params)}`),
};

function toQuery(params?: Record<string, string>): string {
  if (!params || Object.keys(params).length === 0) return "";
  return "?" + new URLSearchParams(params).toString();
}

async function downloadFile(path: string, filename: string) {
  const token = getToken();
  const res = await fetch(`${BASE}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error(`Export failed: ${res.status}`);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// ── Domain types ─────────────────────────────────────────────────────────────

export interface Overview {
  developerCount:       number;
  activeSessionCount:   number;
  violationsToday:      number;
  mcpServersDiscovered: number;
  tokensToday?:         number;
  costTodayUsd?:        number | null;
}

export interface Session {
  sessionId:     string;
  claudeId:      string;
  startedAt:     string;
  endedAt:       string | null;
  duration:      number | null;
  requestCount:  number;
  toolCallCount: number;
  status:        string;
  developer:     { email: string; username: string; displayName: string };
  agentName:     string;
  machineId:     string;
  ide:           { name: string; version: string; workspacePath: string; entrypoint?: string };
  model?:        string;
  usage?:        UsageTotals;   // absent = no token data received (not zero)
}

// ── Token usage ──────────────────────────────────────────────────────────────

export interface UsageTotals {
  requests:            number;
  inputTokens:         number;
  outputTokens:        number;
  cacheReadTokens:     number;
  cacheCreationTokens: number;
  totalTokens:         number;
  costUsd:             number | null;  // null = nothing priced
  unpricedRequests:    number;         // >0 means costUsd is a lower bound
}

export interface SessionUsageDetail {
  hasData:    boolean;
  totals?:    UsageTotals;
  byModel?:   (UsageTotals & { model: string })[];
  subagents?: UsageTotals;
  series?:    { at: string; model: string; tokens: number; cumulativeTokens: number; cumulativeCostUsd: number | null }[];
}

export interface TokenUsageAnalytics {
  days:        number;
  sessions:    number;
  totals:      UsageTotals;
  daily:       (UsageTotals & { date: string })[];
  byModel:     (UsageTotals & { model: string })[];
  byDeveloper: (UsageTotals & { email: string; sessions: number })[];
  topSessions: (UsageTotals & { sessionId: string; email: string; startedAt: string })[];
}

export interface ModelPrice {
  modelPrefix:         string;
  effectiveFrom:       string;
  inputPerMtok:        number;
  outputPerMtok:       number;
  cacheReadPerMtok:    number;
  cacheWrite5mPerMtok: number;
  cacheWrite1hPerMtok: number;
  note:                string;
}

export interface PolicyAction {
  policyActionId:   string;
  actionType:       "ALLOW" | "ALERT" | "BLOCK";
  violationType:    string;
  severity:         "low" | "medium" | "high";
  resourceType:     string;
  resourceId:       string;
  reason:           string;
  matchedRule:      string;
  policyName:       string;
  occurredAt:       string;
  resolutionStatus: string;
  developer?:       { email: string; username: string };
  agentName?:       string;
  machineId?:       string;
  metadata?:        Record<string, unknown> | null;
}

/**
 * How a third-party enforcement verdict was established.
 * "telemetry"        — OS-level ground truth: the process never started, or the
 *                      target file never changed.
 * "hook_correlation" — inferred from a missing PostToolUse marker after the
 *                      correlation timeout, or reported by a cooperating hook.
 */
export type DetectionMethod = "transcript" | "telemetry" | "response_signature" | "hook_correlation";

/**
 * Shape of PolicyAction.metadata for third-party enforcement violations
 * (resourceType === "third_party"). Stored as JSONB server-side, so every field
 * is optional from the client's point of view; detectionMethod in particular is
 * absent on rows ingested before telemetry detection shipped.
 */
export interface GovernanceMetadata {
  toolName?: string;
  eventId?: string;
}

export interface ThirdPartyMetadata {
  thirdPartyTool?:        string;
  /** block | warn | allow | ask | error — what the other tool decided. */
  thirdPartyAction?:      string;
  attributionConfidence?: "definitive" | "candidate";
  detectionMethod?:       DetectionMethod;
  /** The reason the third-party tool gave, verbatim. */
  thirdPartyReason?:      string;
  /** Which hook decided, e.g. "PreToolUse:Bash". */
  thirdPartyHookEvent?:   string;
  blockedClaudeTool?:     string;
  eventId?:               string;
}

export interface Developer {
  developerId:    string;
  email:          string;
  displayName:    string;
  username:       string;
  role:           string;
  status:         string;
  firstSeenAt:    string;
  lastSeenAt:     string;
  violationCount: number;
}

export interface Agent {
  agentId:       string;
  agentName:     string;
  vendor:        string;
  agentType:     string;
  version:       string;
  status:        string;
  firstSeenAt:   string;
  lastSeenAt:    string;
  instanceCount: number;
}

export interface MCPServer {
  mcpServerId:  string;
  serverName:   string;
  serverUrl:    string;
  vendor:       string;
  status:       string;
  firstSeenAt:  string;
  lastSeenAt:   string;
  sessionCount: number;
  totalCalls:   number;
}

/**
 * Four-way control status. Deliberately raw counts, never a percentage — the
 * compliance design (§5.3/§8.1) is explicit that no single "% compliant" number
 * is derived from these, because a partially-evidenced control is not a
 * fraction of a compliant one.
 */
export type ControlStatus = "compliant" | "partially_compliant" | "non_compliant" | "planned";

/** Whether a control is actually implemented, or merely scoped. */
export type EvidenceStatus = "operational" | "planned";

export interface ComplianceFramework {
  frameworkId:             string;
  frameworkType:           string; // "eu_ai_act" | "soc2"
  displayName:             string;
  applicableControlCount:  number;
  compliantCount:          number;
  partiallyCompliantCount: number;
  nonCompliantCount:       number;
  plannedCount:            number;
  /** Null until the scheduler has run its first calculation for this framework. */
  calculatedAt:            string | null;
}

export interface FrameworkControl {
  controlId:            string;
  controlCode:          string;
  /** The code this framework uses for the control, e.g. "CC6.1" vs our internal code. */
  frameworkControlCode: string;
  title:                string;
  description:          string;
  evidenceStatus:       EvidenceStatus;
  currentStatus:        ControlStatus;
  evidenceCount:        number;
}

export interface ControlEvidence {
  evidenceId: string;
  sourceType: "audit_event" | "activity_event" | "policy_action" | "";
  sourceId:   string;
  observedAt: string;
}

export interface AuditEvent {
  auditEventId: string;
  actorType:    string;
  eventType:    string;
  resourceType: string;
  timestamp:    string;
  actorEmail:   string;
  metadata:     Record<string, unknown> | null;
}

export interface PolicyListItem {
  policyId:        string;
  policyName:      string;
  description:     string;
  status:          string;           // "active" | "disabled"
  policyType:      string;           // "governance" | "pii" | "third_party_monitor"
  createdAt:       string;
  updatedAt:       string;
  currentVersionId: string;
  versionNumber:   number;
  enforcementMode: string;
  changeSummary:   string;
  definition:      PackPolicyDef | ThirdPartyMonitorDef | null;
}

export interface ThirdPartyMonitorDef {
  third_party_tool:   string;
  detected_action:    string;
  our_action:         string;        // user-editable: "allow" | "alert" | "block"
  occurrence_count:   number;
  first_seen_at:      string;
}

export interface SessionHook {
  id:        string;
  sessionId: string;
  event:     string;
  matcher:   string;
  command:   string;
  vendor:    string;
  createdAt: string;
}

export interface PIIMatch {
  name:   string;
  action: string;
}

export interface UpdatePolicyBody {
  policyName:      string;
  description:     string;
  definition:      unknown;
  enforcementMode: string;
  changeSummary:   string;
}

export interface PackPolicyDef {
  policyId:     string;
  policyName:   string;
  type:         string;
  action:       string;
  matchTarget:  string;
  pattern:      string;
  agentMessage?: string;
}

export interface PolicyHistoryEntry {
  policyVersionId: string;
  versionNumber:   number;
  enforcementMode: string;
  createdAt:       string;
  changeSummary:   string;
  isActive:        boolean;
}

export interface CreatePolicyBody {
  policyName:      string;
  description:     string;
  definition:      unknown;
  enforcementMode: string;
  changeSummary:   string;
}

export interface ActivateResponse {
  policyId:      string;
  policyVersion: number;
  packVersion:   number;
}

export interface Skill {
  skillId:          string;
  skillName:        string;
  status:           string;
  firstSeenAt:      string;
  lastSeenAt:       string;
  totalInvocations: number;
  sessionCount:     number;
}

export interface SessionSkill {
  skillId:         string;
  skillName:       string;
  status:          string;
  invocationCount: number;
  firstInvokedAt:  string;
  lastInvokedAt:   string;
}

export interface SkillDetail extends Skill {
  sessions: {
    sessionId:       string;
    claudeId:        string;
    invocationCount: number;
    firstInvokedAt:  string;
    lastInvokedAt:   string;
    developer:       string;
    agent:           string;
  }[];
}
