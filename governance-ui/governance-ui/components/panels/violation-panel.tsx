"use client";
// Architecture §9.3 — Violation Detail Panel: context + links (left) + raw JSON (right).

import { useQuery } from "@tanstack/react-query";
import { api, type GovernanceMetadata, type ThirdPartyMetadata } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { ActionBadge } from "@/components/action-badge";
import { DetectionBadge } from "@/components/detection-badge";
import { X, Copy } from "lucide-react";
import React from "react";

interface Props { id: string; onClose: () => void; }

export function ViolationPanel({ id, onClose }: Props) {
  const { data: violations = [] } = useQuery({
    queryKey: ["violations"],
    queryFn:  () => api.violations(),
  });
  const violation = violations.find((v) => v.policyActionId === id);
  const { openPanel } = usePanels();

  const isPII        = violation?.resourceType === "prompt";
  const isThirdParty = violation?.resourceType === "third_party";

  const meta = violation?.metadata ?? {};
  const piiMatches   = (meta.piiMatches  as { name: string; action: string }[] | undefined) ?? [];
  const piiDetected  = !!(meta.piiDetected);
  const tp           = meta as ThirdPartyMetadata;
  const tpTool       = tp.thirdPartyTool        ?? "";
  const tpAction     = tp.thirdPartyAction      ?? "";
  const tpConfidence = tp.attributionConfidence ?? "";
  // Absent on rows ingested before telemetry detection shipped — those were all
  // heuristic by definition, so read them as such rather than showing nothing.
  const tpDetection  = tp.detectionMethod || "hook_correlation";
  const tpReason     = tp.thirdPartyReason    ?? "";
  const tpHookEvent  = tp.thirdPartyHookEvent ?? "";
  const blockedTool  = tp.blockedClaudeTool ?? "";
  const toolName     = (meta as GovernanceMetadata).toolName ?? "";

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-gray-900">
        <div>
          <p className="text-xs text-gray-500">Observe / Violations / {id}</p>
          {violation && (
            <div className="flex items-center gap-2 mt-1">
              <ActionBadge action={violation.actionType} />
              <span className="text-xs text-gray-400">{new Date(violation.occurredAt).toLocaleString()}</span>
            </div>
          )}
        </div>
        <button onClick={onClose} className="text-gray-400 hover:text-white"><X size={18} /></button>
      </div>

      {!violation && (
        <div className="p-8 text-gray-600 text-sm">Loading violation details…</div>
      )}

      {violation && (
        <div className="flex flex-1 overflow-hidden">
          {/* LEFT — context */}
          <div className="w-1/2 border-r border-gray-800 overflow-y-auto p-6 space-y-6">

            {/* Context links */}
            <section>
              <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Context</p>
              <dl className="space-y-2 text-sm">
                <Row label="Developer">
                  <button onClick={() => openPanel("developer", violation.developer?.email ?? "")}
                    className="text-blue-400 hover:underline font-mono">
                    {violation.developer?.email}  →
                  </button>
                </Row>
                {violation.machineId && (
                  <Row label="Machine">
                    <button onClick={() => openPanel("machine", violation.machineId!)}
                      className="text-blue-400 hover:underline font-mono text-xs">
                      {violation.machineId}  →
                    </button>
                  </Row>
                )}
                <Row label="Agent"><span className="text-gray-300">{violation.agentName}</span></Row>
              </dl>
            </section>

            {/* What was matched */}
            <section>
              <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">What was matched</p>
              <dl className="space-y-2 text-sm">
                {!isThirdParty && (
                  <Row label="Tool"><span className="font-mono text-gray-300">{toolName || "—"}</span></Row>
                )}
                <Row label="Category"><span className="text-gray-300">{violation.resourceType}</span></Row>
                <Row label="Matched on"><span className="text-gray-300">{violation.violationType}</span></Row>
                <Row label="Matched value">
                  <span className="font-mono text-xs text-green-400 break-all">{violation.resourceId}</span>
                </Row>
              </dl>
            </section>

            {/* Policy */}
            <section>
              <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Policy that fired</p>
              <dl className="space-y-2 text-sm">
                <Row label="Name">
                  <button onClick={() => openPanel("policy", violation.matchedRule)}
                    className="text-blue-400 hover:underline">
                    {violation.policyName}  →
                  </button>
                </Row>
                <Row label="Severity"><ActionBadge action={violation.actionType} /></Row>
              </dl>
            </section>

            {/* PII detection details */}
            {isPII && piiDetected && piiMatches.length > 0 && (
              <section>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">PII Detected</p>
                <div className="flex flex-wrap gap-1.5 mb-3">
                  {piiMatches.map((m) => (
                    <span
                      key={m.name}
                      className={`px-2 py-0.5 rounded text-xs font-mono ${
                        m.action === "block"
                          ? "bg-red-900/40 text-red-400 border border-red-800"
                          : "bg-amber-900/40 text-amber-300 border border-amber-800"
                      }`}
                    >
                      {m.name} · {m.action}
                    </span>
                  ))}
                </div>
                <p className="text-xs text-gray-600">
                  The prompt was scanned and the matched text was replaced with{" "}
                  <code className="text-amber-400">[PII:type]</code> tokens before storage.
                  Raw prompt content is never stored.
                </p>
              </section>
            )}

            {/* Third-party attribution */}
            {isThirdParty && (
              <section>
                <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Third-Party Enforcement</p>
                <dl className="space-y-2 text-sm">
                  <Row label="Blocked tool">
                    <span className="font-mono text-gray-300">{blockedTool || "—"}</span>
                  </Row>
                  <Row label="Hook">
                    <span className="font-mono text-orange-300">{tpTool || "unknown"}</span>
                  </Row>
                  <Row label="Their action">
                    <span className={`px-2 py-0.5 rounded text-xs font-semibold ${
                      tpAction === "block"
                        ? "bg-red-900/40 text-red-400"
                        : "bg-orange-900/40 text-orange-300"
                    }`}>
                      {(tpAction || "unknown").toUpperCase()}
                    </span>
                  </Row>
                  {tpConfidence && (
                    <Row label="Attribution">
                      <span className="text-gray-400 text-xs">{tpConfidence}</span>
                    </Row>
                  )}
                  {tpHookEvent && (
                    <Row label="Hook event">
                      <span className="font-mono text-xs text-gray-400">{tpHookEvent}</span>
                    </Row>
                  )}
                  <Row label="Detected by">
                    <DetectionBadge method={tpDetection} />
                  </Row>
                </dl>

                {/* The tool's own words, when it gave any. Quoted rather than
                    paraphrased — this is the primary answer to "why". */}
                {tpReason && (
                  <div className="mt-3 border-l-2 border-orange-800 pl-3">
                    <p className="text-[10px] uppercase tracking-wider text-gray-600 mb-1">
                      Reason given by {tpTool || "the tool"}
                    </p>
                    <p className="text-xs text-orange-200 whitespace-pre-wrap break-words">{tpReason}</p>
                  </div>
                )}

                {tpDetection === "transcript" ? (
                  <p className="text-xs text-gray-600 mt-3">
                    Read from the agent transcript: the name above is the command the deciding hook
                    actually ran, and the reason is its own text. Neither is inferred.
                  </p>
                ) : tpDetection === "telemetry" ? (
                  <p className="text-xs text-gray-600 mt-3">
                    Predates a fix: this was reported from an OS-level check finding no evidence the
                    tool ran, which does not actually confirm a block — a fast command can start and
                    exit between polls, looking identical to one that never ran. Newer events with
                    this signal and no transcript confirmation are no longer reported as violations.
                  </p>
                ) : (
                  <p className="text-xs text-gray-600 mt-3">
                    Inferred from a missing <code className="text-amber-400">PostToolUse</code>{" "}
                    marker after the correlation timeout. No OS-level signal was available to
                    confirm it independently.
                  </p>
                )}
              </section>
            )}
          </div>

          {/* RIGHT — raw JSON */}
          <div className="w-1/2 flex flex-col">
            <div className="flex items-center justify-between px-4 py-2 border-b border-gray-800 bg-gray-900">
              <span className="text-xs text-gray-500 uppercase tracking-wider">Full Event Payload</span>
              <button
                onClick={() => navigator.clipboard.writeText(JSON.stringify(violation, null, 2))}
                className="text-xs text-gray-500 hover:text-white flex items-center gap-1"
              >
                <Copy size={11} /> Copy
              </button>
            </div>
            <pre className="flex-1 overflow-auto p-4 text-xs font-mono text-green-300 bg-gray-950">
              {JSON.stringify(violation, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex gap-3">
      <dt className="text-gray-500 w-28 shrink-0">{label}:</dt>
      <dd>{children}</dd>
    </div>
  );
}
