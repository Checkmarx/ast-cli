"use client";
// Architecture §9.3 — Developer Detail Panel (also used for Machine scope).

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { ActionBadge } from "@/components/action-badge";
import { X, Download } from "lucide-react";

interface Props { id: string; onClose: () => void; machineScope?: boolean; }

export function DeveloperPanel({ id, onClose, machineScope }: Props) {
  const { data: sessions = [] } = useQuery({
    queryKey: ["sessions", id],
    queryFn:  () => api.sessions(machineScope ? { machineId: id } : { developerId: id }),
  });
  const { data: violations = [] } = useQuery({
    queryKey: ["violations", id],
    queryFn:  () => api.violations(machineScope ? { machineId: id } : { developerId: id }),
  });
  const { openPanel } = usePanels();

  const blocks = violations.filter((v) => v.actionType === "BLOCK").length;
  const alerts = violations.filter((v) => v.actionType === "ALERT").length;

  return (
    <div className="flex flex-col h-screen">
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900">
        <div>
          <p className="text-xs text-gray-500">Observe / {machineScope ? "Machines" : "Developers"} / {id}</p>
          <p className="text-sm font-semibold text-gray-200 mt-0.5">{id}</p>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={onClose} className="text-gray-400 hover:text-white"><X size={18} /></button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* LEFT — overview */}
        <div className="w-1/2 border-r border-gray-800 overflow-y-auto p-6 space-y-6">
          <section>
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Violation Summary</p>
            <div className="grid grid-cols-3 gap-3">
              {[["Total", violations.length, "text-gray-300"], ["BLOCK", blocks, "text-red-400"], ["ALERT", alerts, "text-yellow-400"]].map(([l, v, c]) => (
                <div key={l as string} className="bg-gray-900 rounded p-3 border border-gray-800">
                  <p className="text-xs text-gray-500">{l}</p>
                  <p className={`text-2xl font-semibold mt-1 ${c}`}>{v}</p>
                </div>
              ))}
            </div>
          </section>

          <section>
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">By Category</p>
            {["filesystem", "shell", "mcp", "prompt", "web_search"].map((cat) => {
              const count = violations.filter((v) => v.resourceType === cat).length;
              if (count === 0) return null;
              return (
                <div key={cat} className="flex items-center gap-3 mb-2">
                  <span className="text-xs text-gray-400 w-24">{cat}</span>
                  <div className="flex-1 bg-gray-800 rounded-full h-1.5">
                    <div
                      className="bg-blue-500 h-1.5 rounded-full"
                      style={{ width: `${Math.min(100, (count / violations.length) * 100)}%` }}
                    />
                  </div>
                  <span className="text-xs text-gray-500 w-6 text-right">{count}</span>
                </div>
              );
            })}
          </section>
        </div>

        {/* RIGHT — recent sessions */}
        <div className="w-1/2 overflow-y-auto">
          <div className="px-5 py-3 border-b border-gray-800 flex items-center justify-between sticky top-0 bg-gray-950">
            <p className="text-xs text-gray-500">Recent Sessions  <span className="text-gray-600">{sessions.length}</span></p>
            <button
              onClick={() => api.exportViolations(machineScope ? { machineId: id } : { developerId: id }, "csv")}
              className="text-xs text-gray-500 hover:text-white flex items-center gap-1"
            >
              <Download size={11} /> Export
            </button>
          </div>
          {sessions.map((s) => (
            <button
              key={s.sessionId}
              onClick={() => openPanel("session", s.sessionId)}
              className="w-full text-left px-5 py-3 border-b border-gray-900 hover:bg-gray-900 flex justify-between items-start"
            >
              <div>
                <p className="text-xs text-gray-400">{new Date(s.startedAt).toLocaleString()}</p>
                <p className="text-xs text-gray-500 mt-0.5">{s.agentName} · {s.status}</p>
              </div>
              <span className="text-gray-600 text-xs">→</span>
            </button>
          ))}
          {sessions.length === 0 && <p className="px-5 py-4 text-xs text-gray-600">No sessions found.</p>}
        </div>
      </div>
    </div>
  );
}
