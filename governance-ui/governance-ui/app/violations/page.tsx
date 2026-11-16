"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, PolicyAction, GovernanceMetadata, ThirdPartyMetadata } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { ActionBadge } from "@/components/action-badge";
import { PageError, EmptyState } from "@/components/page-error";

type FilterType = "all" | "pii" | "checkmarx" | "third_party";

const FILTERS: { key: FilterType; label: string }[] = [
  { key: "all",         label: "All"              },
  { key: "pii",         label: "Prompt PII"       },
  { key: "checkmarx",   label: "Checkmarx Policy" },
  { key: "third_party", label: "Other Tool Policy" },
];

function toolNameOf(v: PolicyAction): string {
  const meta = v.metadata ?? {};
  if (v.resourceType === "third_party") return (meta as ThirdPartyMetadata).blockedClaudeTool || "—";
  return (meta as GovernanceMetadata).toolName || "—";
}

function applyFilter(violations: PolicyAction[], filter: FilterType): PolicyAction[] {
  if (filter === "pii")         return violations.filter((v) => v.resourceType === "prompt");
  if (filter === "third_party") return violations.filter((v) => v.resourceType === "third_party");
  if (filter === "checkmarx")   return violations.filter((v) => v.resourceType !== "prompt" && v.resourceType !== "third_party");
  return violations;
}

export default function ViolationsPage() {
  const [filter, setFilter] = useState<FilterType>("all");

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["violations"],
    queryFn:  () => api.violations(),
  });
  const allViolations: PolicyAction[] = data ?? [];
  const violations = applyFilter(allViolations, filter);
  const { openPanel } = usePanels();

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-lg font-semibold text-gray-200">Violations</h1>
        <span className="text-xs text-gray-600">{violations.length} shown / {allViolations.length} total</span>
      </div>

      {/* Filter chips */}
      <div className="flex gap-2 mb-5">
        {FILTERS.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setFilter(key)}
            className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
              filter === key
                ? "bg-blue-600/30 text-blue-300 border border-blue-700"
                : "bg-gray-800 text-gray-400 border border-gray-700 hover:text-gray-200"
            }`}
          >
            {label}
            <span className="ml-1.5 text-gray-500">
              {applyFilter(allViolations, key).length}
            </span>
          </button>
        ))}
      </div>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && violations.length === 0 && (
        <EmptyState label="violations" />
      )}

      {violations.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Time","Developer","Agent","Tool","Action","Policy","Resource","Matched Value"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {violations.map((v) => (
                <tr
                  key={v.policyActionId}
                  onClick={() => openPanel("violation", v.policyActionId)}
                  className="border-t border-gray-800 hover:bg-gray-900 cursor-pointer"
                >
                  <td className="px-4 py-3 text-gray-500">{new Date(v.occurredAt).toLocaleString()}</td>
                  <td className="px-4 py-3 text-gray-300">{v.developer?.email ?? "—"}</td>
                  <td className="px-4 py-3 text-gray-400">{v.agentName ?? "—"}</td>
                  <td className="px-4 py-3 text-gray-300 font-mono">{toolNameOf(v)}</td>
                  <td className="px-4 py-3"><ActionBadge action={v.actionType} /></td>
                  <td className="px-4 py-3 text-gray-400">{v.policyName || v.reason || "—"}</td>
                  <td className="px-4 py-3 text-gray-500">{v.resourceType}</td>
                  <td className="px-4 py-3 text-gray-500 font-mono truncate max-w-[12rem]">{v.resourceId || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
