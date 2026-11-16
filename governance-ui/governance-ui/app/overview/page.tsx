"use client";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import {
  api, Session, PolicyAction, Developer,
  PolicyListItem, MCPServer, Agent, Skill,
} from "@/lib/api";
import { StatCard }    from "@/components/stat-card";
import { ActionBadge } from "@/components/action-badge";
import { usePanels }   from "@/lib/panel-context";
import { formatTokens, formatUsd } from "@/lib/format";

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1)  return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24)  return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function SummaryPanel({
  title, count, onHeaderClick, children, emptyText,
}: {
  title:         string;
  count:         number;
  onHeaderClick: () => void;
  children:      React.ReactNode;
  emptyText:     string;
}) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg flex flex-col overflow-hidden hover:border-gray-700 transition-colors">
      <button
        onClick={onHeaderClick}
        className="flex items-center justify-between px-4 py-3 border-b border-gray-800 hover:bg-gray-800/50 transition-colors w-full text-left"
      >
        <span className="text-xs font-semibold text-gray-300 uppercase tracking-wider">{title}</span>
        <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded-full">{count}</span>
      </button>
      <div className="flex-1 divide-y divide-gray-800/60">
        {count === 0
          ? <p className="px-4 py-5 text-xs text-gray-600">{emptyText}</p>
          : children}
      </div>
    </div>
  );
}

export default function OverviewPage() {
  const router        = useRouter();
  const { openPanel } = usePanels();

  const { data: overview, isError: ovErr } = useQuery({
    queryKey:        ["overview"],
    queryFn:         api.overview,
    refetchInterval: 15_000,
  });

  const { data: sessionsRaw }   = useQuery({ queryKey: ["sessions"],    queryFn: () => api.sessions() });
  const { data: violationsRaw } = useQuery({ queryKey: ["violations"],  queryFn: () => api.violations() });
  const { data: developersRaw } = useQuery({ queryKey: ["developers"],  queryFn: () => api.developers() });
  const { data: agentsRaw }     = useQuery({ queryKey: ["agents"],      queryFn: api.agents });
  const { data: policiesRaw }   = useQuery({ queryKey: ["policies"],    queryFn: () => api.policies() });
  const { data: mcpRaw }        = useQuery({ queryKey: ["mcp-servers"], queryFn: api.mcpServers });
  const { data: skillsRaw }     = useQuery({ queryKey: ["skills"],      queryFn: api.skills });

  const sessions:   Session[]        = sessionsRaw   ?? [];
  const violations: PolicyAction[]   = violationsRaw ?? [];
  const developers: Developer[]      = developersRaw ?? [];
  const agents:     Agent[]          = (agentsRaw    ?? []).filter((a) => a.instanceCount > 0);
  const policies:   PolicyListItem[] = (policiesRaw  ?? []).filter((p) => p.policyType !== "third_party_monitor");
  const mcpServers: MCPServer[]      = mcpRaw        ?? [];
  const skills:     Skill[]          = skillsRaw     ?? [];

  return (
    <div className="space-y-6">
      {/* Stat cards */}
      <div>
        <h1 className="text-lg font-semibold text-gray-200 mb-4">Overview</h1>
        {ovErr && <p className="text-xs text-red-400 mb-4">API error — backend may be down</p>}
        <div className="grid grid-cols-2 xl:grid-cols-6 gap-4">
          <StatCard label="Developers"       value={overview?.developerCount       ?? "—"} color="blue"   />
          <StatCard label="Active Sessions"  value={overview?.activeSessionCount   ?? "—"} color="green"  />
          <StatCard label="Violations Today" value={overview?.violationsToday      ?? "—"} color="red"    />
          <StatCard label="MCP Servers"      value={overview?.mcpServersDiscovered ?? "—"} color="yellow" />
          <StatCard label="Tokens Today"     value={overview?.tokensToday != null ? formatTokens(overview.tokensToday) : "—"} color="blue"
                    sub="all sessions, since midnight" />
          <StatCard label="Est. Cost Today"  value={formatUsd(overview?.costTodayUsd)} color="green" sub="priced models only" />
        </div>
      </div>

      {/* Summary panels — order: Violations, Sessions, Policies, MCP, Skills, Agents, Developers */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">

        {/* 1. Recent Violations */}
        <SummaryPanel
          title="Recent Violations"
          count={violations.length}
          onHeaderClick={() => router.push("/violations")}
          emptyText="No violations yet. Create a policy to start detecting."
        >
          {violations.slice(0, 4).map((v) => (
            <button
              key={v.policyActionId}
              onClick={() => openPanel("violation", v.policyActionId)}
              className="w-full text-left flex justify-between items-center px-4 py-2.5 hover:bg-gray-800/60 transition-colors"
            >
              <div className="min-w-0">
                <p className="text-xs text-gray-300 truncate">{v.policyName || v.reason || "—"}</p>
                <p className="text-xs text-gray-600 font-mono truncate mt-0.5">{v.resourceId || "—"}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-2">
                <ActionBadge action={v.actionType} />
                <span className="text-gray-700 text-xs">→</span>
              </div>
            </button>
          ))}
        </SummaryPanel>

        {/* 2. Sessions */}
        <SummaryPanel
          title="Recent Sessions"
          count={sessions.length}
          onHeaderClick={() => router.push("/sessions")}
          emptyText="No sessions yet. Start a Claude Code session."
        >
          {sessions.slice(0, 4).map((s) => (
            <button
              key={s.sessionId}
              onClick={() => openPanel("session", s.sessionId)}
              className="w-full text-left flex justify-between items-center px-4 py-2.5 hover:bg-gray-800/60 transition-colors"
            >
              <div className="min-w-0">
                <p className="text-xs text-gray-300 truncate">{s.developer?.email ?? "—"}</p>
                <p className="text-xs text-gray-600 mt-0.5">{s.agentName}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-2">
                <span
                  style={{
                    display: "inline-block", padding: "2px 8px", borderRadius: "10px",
                    fontSize: "11px", fontWeight: 600,
                    ...(s.status === "active"
                      ? { background: "#4dbb40", color: "#fff" }
                      : s.status === "expired"
                      ? { background: "#f9ae4d", color: "#413e48" }
                      : { background: "#939499", color: "#fff" }),
                  }}
                >{s.status === "expired" ? "killed" : s.status}</span>
                <span className="text-gray-700 text-xs">→</span>
              </div>
            </button>
          ))}
        </SummaryPanel>

        {/* 3. Policies */}
        <SummaryPanel
          title="Policies"
          count={policies.length}
          onHeaderClick={() => router.push("/policies")}
          emptyText="No policies. Go to Policies to create one."
        >
          {policies.slice(0, 4).map((p) => (
            <button
              key={p.policyId}
              onClick={() => openPanel("policy", p.policyId)}
              className="w-full text-left flex justify-between items-center px-4 py-2.5 hover:bg-gray-800/60 transition-colors"
            >
              <div className="min-w-0">
                <p className="text-xs text-gray-300 truncate">{p.policyName}</p>
                <p className="text-xs text-gray-600 mt-0.5">{(p.definition as { type?: string } | null)?.type ?? p.policyType ?? "—"}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-2">
                {(p.definition as { action?: string } | null)?.action && (
                  <ActionBadge action={(p.definition as { action?: string }).action!} />
                )}
                <span className="text-gray-700 text-xs">→</span>
              </div>
            </button>
          ))}
        </SummaryPanel>

        {/* 4. MCP Servers */}
        <SummaryPanel
          title="MCP Servers"
          count={mcpServers.length}
          onHeaderClick={() => router.push("/mcp-servers")}
          emptyText="No MCP servers discovered yet."
        >
          {mcpServers.slice(0, 4).map((m) => (
            <div key={m.mcpServerId} className="flex justify-between items-center px-4 py-2.5">
              <div className="min-w-0">
                <p className="text-xs text-gray-300 font-semibold">{m.serverName}</p>
                <p className="text-xs text-gray-600 mt-0.5">First seen {timeAgo(m.firstSeenAt)}</p>
              </div>
              <div className="text-right shrink-0 ml-2">
                <p className="text-xs text-gray-400">{m.totalCalls} calls</p>
                <p className="text-xs text-gray-600">{m.sessionCount} sessions</p>
              </div>
            </div>
          ))}
        </SummaryPanel>

        {/* 5. Skills */}
        <SummaryPanel
          title="Skills"
          count={skills.length}
          onHeaderClick={() => router.push("/skills")}
          emptyText="No skills discovered yet. Invoke a slash command skill."
        >
          {skills.slice(0, 4).map((s) => (
            <div key={s.skillId} className="flex justify-between items-center px-4 py-2.5">
              <div className="min-w-0">
                <p className="text-xs font-mono font-semibold" style={{ color: "#4258c7" }}>{s.skillName}</p>
                <p className="text-xs text-gray-600 mt-0.5">{s.sessionCount} sessions</p>
              </div>
              <span className="text-xs text-gray-400 shrink-0 ml-2">{s.totalInvocations} uses</span>
            </div>
          ))}
        </SummaryPanel>

        {/* 6. AI Coding Agents */}
        <SummaryPanel
          title="AI Coding Agents"
          count={agents.length}
          onHeaderClick={() => router.push("/agents")}
          emptyText="No agent instances registered yet."
        >
          {agents.slice(0, 4).map((a) => (
            <div key={a.agentId} className="flex justify-between items-center px-4 py-2.5">
              <div className="min-w-0">
                <p className="text-xs text-gray-300 font-semibold">{a.agentName}</p>
                <p className="text-xs text-gray-600 mt-0.5">{a.vendor}</p>
              </div>
              <span className="text-xs font-semibold shrink-0 ml-2" style={{ color: "#4258c7" }}>
                {a.instanceCount} {a.instanceCount === 1 ? "instance" : "instances"}
              </span>
            </div>
          ))}
        </SummaryPanel>

        {/* 7. Developers */}
        <SummaryPanel
          title="Developers"
          count={developers.length}
          onHeaderClick={() => router.push("/developers")}
          emptyText="No developers registered yet."
        >
          {developers.slice(0, 4).map((d) => (
            <button
              key={d.developerId}
              onClick={() => openPanel("developer", d.developerId)}
              className="w-full text-left flex justify-between items-center px-4 py-2.5 hover:bg-gray-800/60 transition-colors"
            >
              <div className="min-w-0">
                <p className="text-xs text-gray-300 truncate">{d.email}</p>
                <p className="text-xs text-gray-600 mt-0.5">Last active {timeAgo(d.lastSeenAt)}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-2">
                {d.violationCount > 0 && (
                  <span className="text-xs font-semibold" style={{ color: "#bb2a31" }}>{d.violationCount} violations</span>
                )}
                <span className="text-gray-700 text-xs">→</span>
              </div>
            </button>
          ))}
        </SummaryPanel>

      </div>

      <p className="text-xs text-gray-700">Stat cards auto-refresh every 15 seconds. Click any panel header to see the full list.</p>
    </div>
  );
}
