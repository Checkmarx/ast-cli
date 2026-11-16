"use client";
import { useQuery } from "@tanstack/react-query";
import { api, Developer } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { PageError, EmptyState } from "@/components/page-error";

export default function DevelopersPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["developers"],
    queryFn:  () => api.developers(),
  });
  const devs: Developer[] = data ?? [];
  const { openPanel } = usePanels();

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-gray-200">Developers</h1>
        <span className="text-xs text-gray-600">{devs.length} total</span>
      </div>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && devs.length === 0 && (
        <EmptyState label="developers" />
      )}

      {devs.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Email","Username","Role","First Seen","Last Active","Violations"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {devs.map((d) => (
                <tr
                  key={d.developerId}
                  onClick={() => openPanel("developer", d.developerId)}
                  className="border-t border-gray-800 hover:bg-gray-900 cursor-pointer"
                >
                  <td className="px-4 py-3 text-gray-300">{d.email}</td>
                  <td className="px-4 py-3 text-gray-500 font-mono">{d.username || "—"}</td>
                  <td className="px-4 py-3 text-gray-500">{d.role}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(d.firstSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(d.lastSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-center">
                    {d.violationCount > 0
                      ? <span className="font-semibold" style={{ color: "#bb2a31" }}>{d.violationCount}</span>
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
