"use client";
import { useQuery } from "@tanstack/react-query";
import { api, Session } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { PageError, EmptyState } from "@/components/page-error";
import { IDEBadge } from "@/components/ide-badge";
import { formatTokens, formatUsdPartial } from "@/lib/format";

export default function SessionsPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["sessions"],
    queryFn:  () => api.sessions(),
  });
  const sessions: Session[] = data ?? [];
  const { openPanel } = usePanels();

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-gray-200">Sessions</h1>
        <span className="text-xs text-gray-600">{sessions.length} total</span>
      </div>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && sessions.length === 0 && (
        <EmptyState label="sessions" />
      )}

      {sessions.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Started", "Developer", "Agent", "IDE", "Machine", "Prompts", "Tool Calls", "Tokens", "Cost", "Status"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sessions.map((s) => {
                const hasNoEvents = (s.requestCount ?? 0) === 0 && (s.toolCallCount ?? 0) === 0;
                return (
                <tr
                  key={s.sessionId}
                  onClick={() => openPanel("session", s.sessionId)}
                  className={`border-t border-gray-800 hover:bg-gray-900 cursor-pointer ${hasNoEvents ? "opacity-50" : ""}`}
                  title={hasNoEvents ? "No events recorded for this session yet" : undefined}
                >
                  <td className="px-4 py-3 text-gray-400">{new Date(s.startedAt).toLocaleString()}</td>
                  <td className="px-4 py-3 text-gray-300">{s.developer?.email ?? "—"}</td>
                  <td className="px-4 py-3 text-gray-400">{s.agentName}</td>
                  <td className="px-4 py-3"><IDEBadge ide={s.ide} /></td>
                  <td className="px-4 py-3 text-gray-500 font-mono text-xs truncate max-w-[8rem]">{s.machineId || "—"}</td>
                  <td className="px-4 py-3 text-gray-400 text-center">{s.requestCount ?? 0}</td>
                  <td className="px-4 py-3 text-gray-400 text-center">{s.toolCallCount ?? 0}</td>
                  <td className="px-4 py-3 text-gray-300 text-right tabular-nums"
                      title={s.usage ? `${s.usage.requests} API requests` : "No token data received"}>
                    {formatTokens(s.usage?.totalTokens)}
                  </td>
                  <td className="px-4 py-3 text-gray-300 text-right tabular-nums"
                      title={s.usage && s.usage.unpricedRequests > 0 ? "Lower bound: some requests use an unpriced model" : undefined}>
                    {formatUsdPartial(s.usage?.costUsd, s.usage?.unpricedRequests ?? 0)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5">
                      <span
                        style={{
                          display:       "inline-block",
                          padding:       "3px 10px",
                          borderRadius:  "12px",
                          fontSize:      "11px",
                          fontWeight:    600,
                          letterSpacing: "0.03em",
                          ...(s.status === "active"
                            ? { background: "#4dbb40", color: "#fff" }
                            : s.status === "expired"
                            ? { background: "#f9ae4d", color: "#413e48" }
                            : { background: "#939499", color: "#fff" }),
                        }}
                      >
                        {s.status === "expired" ? "killed" : s.status}
                      </span>
                      {hasNoEvents && s.status === "active" && (
                        <span className="text-xs text-gray-600" title="Session started but no events received yet">pending</span>
                      )}
                    </div>
                  </td>
                </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
