"use client";
import { useQuery } from "@tanstack/react-query";
import { api, MCPServer } from "@/lib/api";
import { PageError, EmptyState } from "@/components/page-error";

export default function MCPServersPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["mcp-servers"],
    queryFn:  api.mcpServers,
  });
  const servers: MCPServer[] = data ?? [];

  return (
    <div>
      <h1 className="text-lg font-semibold text-gray-200 mb-2">MCP Servers</h1>
      <p className="text-xs text-gray-600 mb-6">
        Auto-discovered when Claude Code uses MCP tools. Block/warn policies are managed in the Policies page.
      </p>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && servers.length === 0 && (
        <EmptyState label="MCP servers" />
      )}

      {servers.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Server Name","Vendor","Status","First Seen","Last Seen","Sessions","Tool Calls"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {servers.map((s) => (
                <tr key={s.mcpServerId} className="border-t border-gray-800 hover:bg-gray-900">
                  <td className="px-4 py-3 text-gray-300 font-semibold">{s.serverName}</td>
                  <td className="px-4 py-3 text-gray-500">{s.vendor || "—"}</td>
                  <td className="px-4 py-3">
                    <span
                      className="px-2 py-0.5 rounded text-xs"
                      style={s.status === "active"
                        ? { background: "#e6f9e6", color: "#2d8a2d", border: "1px solid #4dbb40", borderRadius: "3px" }
                        : { background: "#f6f6f6", color: "#939499", border: "1px solid #ebebeb", borderRadius: "3px" }}
                    >
                      {s.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500">{new Date(s.firstSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(s.lastSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-center text-gray-400">{s.sessionCount}</td>
                  <td className="px-4 py-3 text-center text-gray-400">{s.totalCalls}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
