"use client";
// Architecture §9.3 — Policy Detail Panel: config + version history (left) + recent violations (right).

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { ActionBadge } from "@/components/action-badge";
import { X } from "lucide-react";

interface Props { id: string; onClose: () => void; }

export function PolicyPanel({ id, onClose }: Props) {
  const { data: historyRaw } = useQuery({
    queryKey: ["policy-history", id],
    queryFn:  () => api.policyHistory(id),
  });
  const history = historyRaw ?? [];

  const { data: violationsRaw } = useQuery({
    queryKey: ["violations-policy", id],
    queryFn:  () => api.violations({ policyId: id }),
  });
  const violations = violationsRaw ?? [];
  const { openPanel } = usePanels();

  const active = history.find((h) => h.isActive);

  return (
    <div className="flex flex-col h-screen">
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900">
        <div>
          <p className="text-xs text-gray-500">Observe / Policies / {id}</p>
          {active && (
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm text-gray-200">{active.enforcementMode?.toUpperCase()}</span>
              <span className="text-xs text-gray-500">v{active.versionNumber} (active)</span>
            </div>
          )}
        </div>
        <button onClick={onClose} className="text-gray-400 hover:text-white"><X size={18} /></button>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* LEFT — policy config + version history */}
        <div className="w-1/2 border-r border-gray-800 overflow-y-auto p-6 space-y-6">
          <section>
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Version History</p>
            {history.map((v) => (
              <div
                key={v.policyVersionId}
                className={`flex justify-between items-start px-3 py-2.5 mb-1 rounded border ${
                  v.isActive ? "border-blue-700 bg-blue-900/10" : "border-gray-800"
                }`}
              >
                <div>
                  <span className="text-xs font-semibold text-gray-300">v{v.versionNumber}</span>
                  {v.isActive && <span className="ml-2 text-xs text-blue-400">← active</span>}
                  {v.changeSummary && <p className="text-xs text-gray-500 mt-0.5">"{v.changeSummary}"</p>}
                </div>
                <span className="text-xs text-gray-600">{new Date(v.createdAt).toLocaleDateString()}</span>
              </div>
            ))}
          </section>
        </div>

        {/* RIGHT — recent violations */}
        <div className="w-1/2 overflow-y-auto">
          <div className="px-5 py-3 border-b border-gray-800 sticky top-0 bg-gray-950">
            <p className="text-xs text-gray-500">Recent Violations  <span className="text-gray-600">{violations.length}</span></p>
          </div>
          {violations.map((v) => (
            <button
              key={v.policyActionId}
              onClick={() => openPanel("violation", v.policyActionId)}
              className="w-full text-left px-5 py-3 border-b border-gray-900 hover:bg-gray-900 flex justify-between items-start"
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-300">{v.developer?.email}</span>
                  <span className="text-xs text-gray-500 font-mono">{v.machineId}</span>
                </div>
                <p className="text-xs font-mono text-gray-600 truncate max-w-xs mt-0.5">{v.resourceId}</p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <span className="text-xs text-gray-500">{new Date(v.occurredAt).toLocaleTimeString()}</span>
                <ActionBadge action={v.actionType} />
                <span className="text-gray-600">→</span>
              </div>
            </button>
          ))}
          {violations.length === 0 && <p className="px-5 py-4 text-xs text-gray-600">No violations for this policy.</p>}
        </div>
      </div>
    </div>
  );
}
