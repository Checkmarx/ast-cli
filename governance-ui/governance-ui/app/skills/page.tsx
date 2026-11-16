"use client";
import { useQuery } from "@tanstack/react-query";
import { api, Skill } from "@/lib/api";
import { PageError, EmptyState } from "@/components/page-error";

export default function SkillsPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey:        ["skills"],
    queryFn:         api.skills,
    refetchInterval: 5000,
  });
  const skills: Skill[] = data ?? [];

  return (
    <div>
      <h1 className="text-lg font-semibold text-gray-200 mb-2">Skills Inventory</h1>
      <p className="text-xs text-gray-600 mb-6">
        Auto-discovered when Claude Code invokes a slash command skill. Each row is a unique skill name with usage metrics.
      </p>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && skills.length === 0 && (
        <EmptyState label="skills" />
      )}

      {skills.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Skill Name", "Status", "Total Invocations", "Sessions", "First Seen", "Last Seen"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {skills.map((s) => (
                <tr key={s.skillId} className="border-t border-gray-800 hover:bg-gray-900">
                  <td className="px-4 py-3 font-mono font-semibold" style={{ color: "#4258c7" }}>{s.skillName}</td>
                  <td className="px-4 py-3">
                    <span
                      className="px-2 py-0.5 rounded text-xs"
                      style={
                        s.status === "active"     ? { background: "#e6f9e6", color: "#2d8a2d", border: "1px solid #4dbb40", borderRadius: "3px" } :
                        s.status === "deprecated" ? { background: "#fff3dc", color: "#b07400", border: "1px solid #f9ae4d", borderRadius: "3px" } :
                                                    { background: "#fde8e8", color: "#bb2a31", border: "1px solid #f1605d", borderRadius: "3px" }
                      }
                    >
                      {s.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-center text-gray-300 font-semibold">{s.totalInvocations}</td>
                  <td className="px-4 py-3 text-center text-gray-400">{s.sessionCount}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(s.firstSeenAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-gray-500">{new Date(s.lastSeenAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
