"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { PolicyListItem, ThirdPartyMonitorDef, UpdatePolicyBody } from "@/lib/api";

interface Props {
  policy:   PolicyListItem;
  onSave:   (body: UpdatePolicyBody) => void;
  onClose:  () => void;
  loading:  boolean;
  error:    string | null;
}

const ACTIONS   = ["ALERT", "BLOCK"] as const;
const OUR_ACTIONS = ["allow", "alert", "block"] as const;
const INPUT = "w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-600 transition-colors";

export function EditPolicyModal({ policy, onSave, onClose, loading, error }: Props) {
  const isThirdParty = policy.policyType === "third_party_monitor";
  const tpDef = isThirdParty ? (policy.definition as ThirdPartyMonitorDef | null) : null;

  const [name,    setName]    = useState(policy.policyName);
  const [desc,    setDesc]    = useState(policy.description ?? "");
  const [action,  setAction]  = useState(() => {
    if (isThirdParty) return tpDef?.our_action ?? "alert";
    const def = policy.definition as { action?: string } | null;
    return def?.action ?? "ALERT";
  });
  const [pattern,  setPattern]  = useState(() => {
    if (isThirdParty) return "";
    const def = policy.definition as { pattern?: string } | null;
    return def?.pattern ?? "";
  });
  const [message,  setMessage]  = useState(() => {
    if (isThirdParty) return "";
    const def = policy.definition as { agentMessage?: string } | null;
    return def?.agentMessage ?? "";
  });
  const [summary, setSummary] = useState("");

  const handleSave = () => {
    let definition: unknown;
    if (isThirdParty) {
      definition = { ...(tpDef ?? {}), our_action: action };
    } else {
      const existingDef = policy.definition as Record<string, unknown> | null;
      definition = { ...(existingDef ?? {}), action, pattern, agentMessage: message };
    }
    onSave({
      policyName:      name.trim(),
      description:     desc.trim(),
      definition,
      enforcementMode: action.toLowerCase(),
      changeSummary:   summary.trim() || "Updated via dashboard",
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-lg mx-4 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <h2 className="text-sm font-semibold text-gray-200">Edit Policy</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-white"><X size={16} /></button>
        </div>

        <div className="px-6 py-5 space-y-4">
          {error && (
            <div className="bg-red-950/40 border border-red-800 rounded p-2 text-xs text-red-400">{error}</div>
          )}

          {/* Name */}
          <div>
            <label className="text-xs text-gray-500 block mb-1">Policy Name</label>
            <input value={name} onChange={(e) => setName(e.target.value)} className={INPUT} />
          </div>

          {/* Description */}
          <div>
            <label className="text-xs text-gray-500 block mb-1">Description</label>
            <input value={desc} onChange={(e) => setDesc(e.target.value)} className={INPUT} />
          </div>

          {/* Third-party monitor: only our_action is editable */}
          {isThirdParty ? (
            <div>
              <label className="text-xs text-gray-500 block mb-1">Our Response Action</label>
              <select value={action} onChange={(e) => setAction(e.target.value)} className={INPUT}>
                {OUR_ACTIONS.map((a) => <option key={a} value={a}>{a.toUpperCase()}</option>)}
              </select>
              <p className="text-xs text-gray-600 mt-1">
                This is what our governance system records when this third-party tool is detected.
                The third-party tool&apos;s own action cannot be changed.
              </p>
              {tpDef && (
                <div className="mt-3 bg-gray-800 rounded p-3 text-xs text-gray-500 space-y-1">
                  <p><span className="text-gray-400">Tool:</span> {tpDef.third_party_tool}</p>
                  <p><span className="text-gray-400">Detected action:</span> {tpDef.detected_action}</p>
                  <p><span className="text-gray-400">Occurrences:</span> {tpDef.occurrence_count}</p>
                </div>
              )}
            </div>
          ) : (
            <>
              {/* Action */}
              <div>
                <label className="text-xs text-gray-500 block mb-1">Action</label>
                <select value={action} onChange={(e) => setAction(e.target.value)} className={INPUT}>
                  {ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
                </select>
              </div>

              {/* Pattern */}
              <div>
                <label className="text-xs text-gray-500 block mb-1">Regex Pattern</label>
                <input value={pattern} onChange={(e) => setPattern(e.target.value)} className={INPUT + " font-mono"} />
              </div>

              {/* Agent message */}
              <div>
                <label className="text-xs text-gray-500 block mb-1">Message shown to developer</label>
                <input value={message} onChange={(e) => setMessage(e.target.value)} className={INPUT} />
              </div>
            </>
          )}

          {/* Change summary */}
          <div>
            <label className="text-xs text-gray-500 block mb-1">Change Summary (optional)</label>
            <input
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="e.g. Updated pattern to include .env.local"
              className={INPUT}
            />
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-800">
          <button onClick={onClose} className="text-xs text-gray-400 hover:text-white px-3 py-1.5">
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={loading || !name.trim()}
            className="text-xs px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-40 text-white rounded transition-colors"
          >
            {loading ? "Saving…" : "Save & Activate"}
          </button>
        </div>
      </div>
    </div>
  );
}
