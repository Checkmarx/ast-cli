"use client";
// Architecture §9.3 — Session Detail Panel.
// Left: React Flow node graph (category breakdown) + Findings list.
// Right: Activity timeline with icon, action, content preview, expand.

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

// Shape returned by GET /api/v1/sessions/:id/timeline (activity_events table)
interface TimelineEvent {
  activityEventId: string;
  eventId:         string;
  kind:            string;
  agentType:       string;
  tool: {
    category: string;
    name:     string;
    command:  string;
    query:    string;
    paths:    string[] | null;
    mcp:      { server: string; name: string };
  };
  promptLength:          number | null;
  piiDetected:           boolean;
  promptText:            string | null;
  piiMatches:            { name: string; action: string }[] | null;
  thirdPartyTool:        string | null;
  thirdPartyAction:      string | null;
  attributionConfidence: string | null;
  detectionMethod:       string | null;
  thirdPartyReason:      string | null;
  thirdPartyHookEvent:   string | null;
  decision: {
    action:       string;
    policyName:   string;
    policyId:     string;
    matchedOn:    string;
    matchedValue: string;
    agentMessage: string;
    latencyMs:    number | null;
  };
  occurredAt: string;
}
import { usePanels } from "@/lib/panel-context";
import { ActionBadge } from "@/components/action-badge";
import { DetectionBadge } from "@/components/detection-badge";
import { TokenUsageSection } from "@/components/token-usage-section";
import { IDEBadge } from "@/components/ide-badge";
import { useState, useCallback } from "react";
import { X, Download } from "lucide-react";
import {
  ReactFlow, Background, Controls,
  type Node, type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

interface Props { id: string; onClose: () => void; }

// Icon by category/kind — architecture §9.3 right-panel icons
const ICON: Record<string, string> = {
  "session.start": "⚡", "session.end": "⚡",
  "prompt.submit": "○",
  "filesystem": "✎", "shell": ">_", "mcp": "⬡", "web_search": "🔍",
  "skill": "✦",
};

// Architecture §9.3 category label map
const CAT_LABEL: Record<string, string> = {
  filesystem: "Files", shell: "Bash", mcp: "MCP",
  web_search: "Web", prompt: "Prompts",
};

export function SessionPanel({ id, onClose }: Props) {
  const { data, isLoading, isError } = useQuery({
    queryKey:        ["session-timeline", id],
    queryFn:         () => api.sessionTimeline(id),
    refetchInterval: 5000,
  });
  const { data: skillsData } = useQuery({
    queryKey:        ["session-skills", id],
    queryFn:         () => api.sessionSkills(id),
    refetchInterval: 5000,
  });
  // Fetch sessions to get IDE + workspace metadata (served from cache if sessions page visited first).
  const { data: sessionsData } = useQuery({
    queryKey: ["sessions"],
    queryFn:  () => api.sessions(),
    staleTime: 30_000,
  });
  const sessionMeta = (sessionsData ?? []).find((s) => s.sessionId === id);
  const ide = sessionMeta?.ide;

  const events: TimelineEvent[] = ((data as unknown) as TimelineEvent[] | null) ?? [];
  const skills = skillsData ?? [];
  const { openPanel } = usePanels();
  const findings = events.filter((e) => e.decision?.action === "BLOCK" || e.decision?.action === "ALERT");

  // Build React Flow nodes + edges from events (architecture §9.3)
  const { nodes, edges } = buildSessionGraph(events, id);

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900 shrink-0">
        <div>
          <p className="text-xs text-gray-500">Observe / Sessions / {id}</p>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-xs font-mono text-gray-400 truncate max-w-sm">{id}</p>
            {/* Only once the session's metadata is loaded; otherwise "Unknown" would be wrong. */}
            {sessionMeta && <IDEBadge ide={ide} showVersion />}
            {ide?.workspacePath && (
              <span className="text-xs text-gray-600 font-mono truncate max-w-xs" title={ide.workspacePath}>
                {ide.workspacePath.split(/[\\/]/).pop()}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={() => api.exportSession(id, "csv")}
            className="text-xs text-gray-400 hover:text-white flex items-center gap-1">
            <Download size={11} /> CSV
          </button>
          <button onClick={() => api.exportSession(id, "xlsx")}
            className="text-xs text-gray-400 hover:text-white flex items-center gap-1">
            <Download size={11} /> Excel
          </button>
          <button onClick={onClose} className="text-gray-400 hover:text-white ml-2"><X size={18} /></button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden min-h-0">
        {/* LEFT panel — graph + findings */}
        <div className="w-1/2 border-r border-gray-800 flex flex-col overflow-hidden">
          {/* React Flow graph — architecture §9.3 */}
          <div className="h-64 border-b border-gray-800 bg-gray-950">
            <ReactFlow
              nodes={nodes}
              edges={edges}
              fitView
              proOptions={{ hideAttribution: true }}
              nodesDraggable={false}
              nodesConnectable={false}
              zoomOnScroll={false}
              panOnDrag={false}
            >
              <Background color="#1f2937" gap={20} />
              <Controls showInteractive={false} />
            </ReactFlow>
          </div>

          {/* Token usage + Findings + Skills */}
          <div className="flex-1 overflow-y-auto p-4">
            <TokenUsageSection sessionId={id} />
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">
              Findings{" "}
              <span className="ml-1 px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">{findings.length}</span>
            </p>
            {findings.length === 0 && !isLoading && (
              <p className="text-xs text-gray-600">No findings for this session.</p>
            )}
            {findings.map((f) => (
              <button
                key={f.activityEventId}
                onClick={() => openPanel("violation", f.activityEventId)}
                className="w-full text-left flex justify-between items-start px-3 py-2.5 mb-1.5 rounded
                           hover:bg-gray-800 border border-gray-800"
              >
                <div className="min-w-0">
                  <ActionBadge action={f.decision?.action ?? ""} />
                  <p className="text-xs text-gray-400 mt-1 truncate">{f.decision?.policyName || f.kind}</p>
                  <p className="text-xs text-gray-600 font-mono truncate">{f.decision?.matchedValue || f.tool?.command || ""}</p>
                </div>
                <span className="text-gray-600 text-xs shrink-0 ml-2">
                  {new Date(f.occurredAt).toLocaleTimeString()} →
                </span>
              </button>
            ))}

            {/* Skills section */}
            <p className="text-xs text-gray-500 uppercase tracking-wider mt-5 mb-3">
              Skills Used{" "}
              <span className="ml-1 px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">{skills.length}</span>
            </p>
            {skills.length === 0 && (
              <p className="text-xs text-gray-600">No skills invoked in this session.</p>
            )}
            {skills.map((sk) => (
              <div key={sk.skillId}
                className="px-3 py-2.5 mb-1.5 rounded border border-blue-900/40 bg-blue-950/10">
                <div className="flex justify-between items-center">
                  <span className="text-xs font-mono text-blue-300">{sk.skillName}</span>
                  <span className="text-xs text-gray-500">{sk.invocationCount}×</span>
                </div>
                <p className="text-xs text-gray-600 mt-0.5">
                  {new Date(sk.lastInvokedAt).toLocaleTimeString()}
                </p>
              </div>
            ))}
          </div>
        </div>

        {/* RIGHT panel — activity timeline */}
        <div className="w-1/2 flex flex-col overflow-hidden">
          <div className="px-5 py-3 border-b border-gray-800 sticky top-0 bg-gray-950 shrink-0">
            <p className="text-xs text-gray-500">
              Session Activity{" "}
              <span className="text-gray-600 ml-1">{events.length} actions</span>
            </p>
          </div>
          <div className="flex-1 overflow-y-auto">
            {isLoading && (
              <p className="text-xs text-gray-600 px-5 py-4">Loading timeline…</p>
            )}
            {isError && (
              <div className="px-5 py-4">
                <p className="text-xs text-red-400 font-medium mb-1">Failed to load timeline</p>
                <p className="text-xs text-gray-600">Check that the backend is running on port 8080.</p>
              </div>
            )}
            {!isLoading && !isError && events.length === 0 && (
              <div className="px-5 py-8 text-center">
                <p className="text-sm text-gray-500 mb-1">No events recorded yet</p>
                <p className="text-xs text-gray-600">
                  Events appear here as Claude Code uses tools and submits prompts.
                  <br />If this session just started, events arrive within a few seconds.
                </p>
              </div>
            )}
            {events.map((ev) => (
              <ActivityRow key={ev.activityEventId} event={ev} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── React Flow graph builder — architecture §9.3 ──────────────────────────

function buildSessionGraph(events: TimelineEvent[], sessionId: string): { nodes: Node[]; edges: Edge[] } {
  // Group events by tool category
  const byCat: Record<string, TimelineEvent[]> = {};
  for (const ev of events) {
    const cat = ev.tool?.category;
    if (!cat || cat === "") continue;
    if (!byCat[cat]) byCat[cat] = [];
    byCat[cat].push(ev);
  }

  const agentNode: Node = {
    id: "agent",
    position: { x: 220, y: 80 },
    data: { label: "Claude Code" },
    style: { background: "#1e3a5f", border: "1px solid #3b82f6", color: "#93c5fd", borderRadius: 8, fontSize: 11, padding: "6px 12px" },
  };

  const catNodes: Node[] = Object.entries(byCat).map(([cat, evs], i) => {
    const hasBlock = evs.some((e) => e.decision?.action === "BLOCK");
    const hasAlert = evs.some((e) => e.decision?.action === "ALERT");
    const borderColor = hasBlock ? "#ef4444" : hasAlert ? "#eab308" : "#374151";
    return {
      id: cat,
      position: { x: 80 + i * 130, y: 220 },
      data: { label: `${CAT_LABEL[cat] ?? cat}\n${evs.length} calls` },
      style: { background: "#111827", border: `1.5px solid ${borderColor}`, color: "#9ca3af", borderRadius: 8, fontSize: 10, padding: "4px 10px", whiteSpace: "pre-line" },
    };
  });

  const edges: Edge[] = catNodes.map((n) => ({
    id: `agent-${n.id}`,
    source: "agent",
    target: n.id,
    style: { stroke: "#374151" },
    animated: false,
  }));

  return { nodes: [agentNode, ...catNodes], edges };
}

// ── Activity row component — architecture §9.3 right panel ────────────────

function ActivityRow({ event }: { event: TimelineEvent }) {
  const [expanded, setExpanded] = useState(false);
  const action = event.decision?.action ?? "";
  const isThirdParty = event.kind === "tool.third_party_enforcement";
  // Only the transcript path sets a real verdict (Decision.Action = "BLOCK").
  // A missing PostToolUse marker with no transcript confirmation is recorded
  // for visibility but asserts nothing — treat it as a neutral gap, not a block.
  const hasVerdict = isThirdParty && action === "BLOCK";
  const isUnconfirmedGap = isThirdParty && !hasVerdict;

  const borderClass =
    action === "BLOCK" ? "border-l-2 border-red-500 bg-red-950/10" :
    action === "ALERT" ? "border-l-2 border-yellow-500 bg-yellow-950/10" :
    isUnconfirmedGap ? "border-l-2 border-gray-700" : "";

  const icon = isThirdParty ? "⚡" : (ICON[event.tool?.category ?? ""] ?? ICON[event.kind] ?? "○");
  const filePath = event.tool?.category === "filesystem" ? (event.tool?.paths?.[0] ?? "") : "";

  let preview: string;
  if (hasVerdict) {
    // Show what was blocked (the Claude tool) and the action taken.
    const blocked = event.tool?.name;
    const actionLabel = (event.thirdPartyAction || "block").toUpperCase();
    preview = blocked ? `${actionLabel} → ${blocked}` : (event.decision?.agentMessage || actionLabel);
  } else if (isUnconfirmedGap) {
    preview = event.decision?.agentMessage || "no completion signal observed";
  } else {
    preview = filePath ||
      event.tool?.command || event.tool?.query ||
      (event.tool?.mcp?.name ? `${event.tool.mcp.server}/${event.tool.mcp.name}` : "") ||
      event.decision?.matchedValue || "";
  }

  return (
    <div className={`px-4 py-3 border-b border-gray-900 ${borderClass} hover:bg-gray-900/50`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-gray-500 shrink-0">{icon}</span>
          <span className="text-xs font-semibold text-gray-300 shrink-0">
            {hasVerdict
              ? (event.thirdPartyTool || "Third-Party Hook")
              : isUnconfirmedGap
              ? (event.tool?.name || "Unconfirmed gap")
              : (event.tool?.name || event.kind)}
          </span>
          <span className="text-gray-600 text-xs truncate font-mono">{preview}</span>
        </div>
        <button
          onClick={() => setExpanded((v) => !v)}
          className="text-gray-600 hover:text-gray-400 shrink-0 text-xs px-1"
        >
          ↗
        </button>
      </div>

      {/* Third-party enforcement card — a real, transcript-confirmed block */}
      {hasVerdict && (
        <div className="mt-2 ml-6 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="px-2 py-0.5 rounded text-xs bg-orange-900/40 text-orange-400 border border-orange-800">
              {(event.thirdPartyAction || "block").toUpperCase()}
            </span>
            {event.tool?.name && (
              <span className="text-xs font-mono text-gray-300">{event.tool.name}</span>
            )}
            <span className="text-xs text-gray-500">by</span>
            <span className="font-mono text-xs text-orange-300">{event.thirdPartyTool}</span>
            {event.attributionConfidence && (
              <span className="text-xs text-gray-600">({event.attributionConfidence})</span>
            )}
            <DetectionBadge method={event.detectionMethod} />
            {event.thirdPartyHookEvent && (
              <span className="text-xs font-mono text-gray-600">{event.thirdPartyHookEvent}</span>
            )}
          </div>
          {/* Prefer the third party's own words; agentMessage is our wording. */}
          {event.thirdPartyReason && (
            <p className="text-xs text-orange-200/80 border-l-2 border-orange-900 pl-2 whitespace-pre-wrap break-words">
              {event.thirdPartyReason}
            </p>
          )}
          <div className="flex items-center gap-1.5 text-xs text-gray-500">
            <span>Our response:</span>
            <ActionBadge action={action} />
          </div>
        </div>
      )}

      {/* Unconfirmed completion gap — a PostToolUse marker never arrived, but
          nothing (transcript or otherwise) confirms who blocked it or that
          anything did; not a violation, shown for debugging visibility only. */}
      {isUnconfirmedGap && (
        <div className="mt-2 ml-6 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="px-2 py-0.5 rounded text-xs bg-gray-800 text-gray-400 border border-gray-700">
              UNCONFIRMED
            </span>
            <DetectionBadge method={event.detectionMethod} />
          </div>
          <p className="text-xs text-gray-600 font-mono">{event.decision?.agentMessage}</p>
        </div>
      )}

      {/* Our own governance action */}
      {!isThirdParty && action && action !== "ALLOW" && (
        <div className="mt-1 ml-6 flex items-center gap-2">
          <ActionBadge action={action} />
          <span className="text-xs text-gray-500">{event.decision?.policyName}</span>
        </div>
      )}

      {/* PII detected prompt */}
      {event.piiDetected && event.promptText && (
        <div className="mt-2 ml-6">
          <p className="text-xs text-gray-500 mb-1">Prompt (PII redacted):</p>
          <p className="text-xs font-mono bg-gray-900 rounded px-3 py-2 text-gray-300 break-all">
            {renderRedactedPrompt(event.promptText)}
          </p>
          {event.piiMatches && event.piiMatches.length > 0 && (
            <div className="flex gap-1 flex-wrap mt-1">
              {event.piiMatches.map((m) => (
                <span key={m.name} className={`px-1.5 py-0.5 rounded text-xs font-mono ${
                  m.action === "block" ? "bg-red-900/30 text-red-400" : "bg-yellow-900/30 text-yellow-400"
                }`}>
                  {m.name}
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      {expanded && (
        <pre className="mt-2 text-xs bg-gray-900 rounded p-3 overflow-x-auto max-h-48 text-green-300 font-mono">
          {JSON.stringify(event, null, 2)}
        </pre>
      )}
    </div>
  );
}

// Renders a redacted prompt string, highlighting [PII:type] tokens in amber.
function renderRedactedPrompt(text: string): React.ReactNode {
  const parts = text.split(/(\[PII:[^\]]+\])/g);
  return parts.map((part, i) =>
    part.startsWith("[PII:") ? (
      <span key={i} className="bg-amber-900/40 text-amber-300 rounded px-0.5">{part}</span>
    ) : (
      <span key={i}>{part}</span>
    )
  );
}
