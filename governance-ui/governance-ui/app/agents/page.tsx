"use client";
import { useQuery } from "@tanstack/react-query";
import { api, Agent } from "@/lib/api";
import { PageError, EmptyState } from "@/components/page-error";

export default function AgentsPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["agents"],
    queryFn:  api.agents,
  });
  // Only show agents that have at least one registered instance (no catalog-only dummy entries).
  const agents: Agent[] = (data ?? []).filter((a) => a.instanceCount > 0);

  return (
    <div>
      <h1 className="text-lg font-semibold text-gray-200 mb-6">AI Coding Agents</h1>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && agents.length === 0 && (
        <EmptyState label="agents" />
      )}

      {agents.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Agent","Vendor","Type","Status","First Seen","Last Seen","Instances"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {agents.map((a) => (
                <tr key={a.agentId} className="border-t border-gray-800 hover:bg-gray-900">
                  <td className="px-4 py-3 text-gray-300 font-semibold">{a.agentName}</td>
                  <td className="px-4 py-3 text-gray-500">{a.vendor || "—"}</td>
                  <td className="px-4 py-3 text-gray-500">{a.agentType || "—"}</td>
                  <td className="px-4 py-3">
                    <span
                      className="px-2 py-0.5 rounded text-xs"
                      style={a.status === "active"
                        ? { background: "#e6f9e6", color: "#2d8a2d", border: "1px solid #4dbb40", borderRadius: "3px" }
                        : { background: "#f6f6f6", color: "#939499", border: "1px solid #ebebeb", borderRadius: "3px" }}
                    >
                      {a.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500">{new Date(a.firstSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(a.lastSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-center">
                    {a.instanceCount > 0
                      ? <span className="font-semibold" style={{ color: "#4258c7" }}>{a.instanceCount}</span>
                      : <span className="text-gray-600">0</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
