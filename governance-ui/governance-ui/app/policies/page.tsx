"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, PolicyListItem, ThirdPartyMonitorDef, UpdatePolicyBody } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { PolicyStatusBadge } from "@/components/policy-status-badge";
import { EditPolicyModal } from "@/components/edit-policy-modal";
import { Plus, ChevronUp, CheckCircle, Pencil, PowerOff, Power, Trash2 } from "lucide-react";

// ── action badge ──────────────────────────────────────────────────────────
const ACTION_INLINE_STYLE: Record<string, React.CSSProperties> = {
  BLOCK: { background: "#fde8e8", color: "#bb2a31", border: "1px solid #f1605d", borderRadius: "3px" },
  ALERT: { background: "#fff3dc", color: "#b07400", border: "1px solid #f9ae4d", borderRadius: "3px" },
  ALLOW: { background: "#f6f6f6", color: "#939499", border: "1px solid #ebebeb", borderRadius: "3px" },
};
function ActionBadge({ action }: { action: string }) {
  const key = action.toUpperCase();
  const style = ACTION_INLINE_STYLE[key] ?? { background: "#f6f6f6", color: "#939499", border: "1px solid #ebebeb", borderRadius: "3px" };
  return (
    <span className="px-2 py-0.5 rounded text-xs font-semibold" style={style}>
      {key}
    </span>
  );
}

// ── filter tabs ───────────────────────────────────────────────────────────
type FilterTab = "governance" | "disabled";
const TABS: { key: FilterTab; label: string }[] = [
  { key: "governance", label: "Checkmarx Policy" },
  { key: "disabled",   label: "Disabled"         },
];

export default function PoliciesPage() {
  const [showCreate,  setShowCreate]  = useState(false);
  const [activeTab,   setActiveTab]   = useState<FilterTab>("governance");
  const [editPolicy,  setEditPolicy]  = useState<PolicyListItem | null>(null);
  const [confirmDel,  setConfirmDel]  = useState<PolicyListItem | null>(null);
  const qc = useQueryClient();
  const { openPanel } = usePanels();

  const { data: allPolicies = [], isLoading, error } = useQuery({
    queryKey: ["policies"],
    queryFn:  () => api.policies({ status: "all" }),
  });

  // Only governance-type policies shown; third_party_monitor and pii excluded.
  const visiblePolicies = allPolicies.filter((p) => p.policyType === "governance" || p.status === "disabled");
  const policies = visiblePolicies.filter((p) =>
    activeTab === "disabled"
      ? p.status === "disabled"
      : p.policyType === "governance" && p.status !== "disabled"
  );

  const createMutation = useMutation({
    mutationFn: async (body: Parameters<typeof api.createPolicy>[0]) => {
      const created = await api.createPolicy(body);
      await api.activatePolicy(created.policyId, { policyVersionId: created.policyVersionId });
      return created;
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["policies"] }); setShowCreate(false); },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdatePolicyBody }) => api.updatePolicy(id, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["policies"] }); setEditPolicy(null); },
  });

  const activateMutation = useMutation({
    mutationFn: ({ id, pvId }: { id: string; pvId: string }) => api.activatePolicy(id, { policyVersionId: pvId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["policies"] }),
  });

  const disableMutation = useMutation({
    mutationFn: (id: string) => api.disablePolicy(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["policies"] }),
  });

  const enableMutation = useMutation({
    mutationFn: (id: string) => api.enablePolicy(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["policies"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deletePolicy(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["policies"] }); setConfirmDel(null); },
  });

  return (
    <div>
      {/* Edit modal */}
      {editPolicy && (
        <EditPolicyModal
          policy={editPolicy}
          loading={updateMutation.isPending}
          error={updateMutation.error ? String(updateMutation.error) : null}
          onClose={() => setEditPolicy(null)}
          onSave={(body) => updateMutation.mutate({ id: editPolicy.policyId, body })}
        />
      )}

      {/* Delete confirm modal */}
      {confirmDel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-sm mx-4 p-6 shadow-2xl">
            <h3 className="text-sm font-semibold text-gray-200 mb-2">Delete Policy</h3>
            <p className="text-xs text-gray-400 mb-4">
              <span className="font-semibold text-gray-200">{confirmDel.policyName}</span> will be
              permanently deleted. Historical violations will retain a reference to it.
              This cannot be undone.
            </p>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setConfirmDel(null)} className="text-xs text-gray-400 hover:text-white px-3 py-1.5">
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate(confirmDel.policyId)}
                disabled={deleteMutation.isPending}
                className="text-xs px-4 py-1.5 bg-red-700 hover:bg-red-600 disabled:opacity-40 text-white rounded"
              >
                {deleteMutation.isPending ? "Deleting…" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-lg font-semibold text-gray-200">Policies</h1>
        <button
          onClick={() => setShowCreate((v) => !v)}
          className="text-xs text-blue-400 border border-blue-700 px-3 py-1.5 rounded flex items-center gap-1.5 hover:bg-blue-900/20 transition-colors"
        >
          {showCreate ? <ChevronUp size={12} /> : <Plus size={12} />}
          {showCreate ? "Cancel" : "New Policy"}
        </button>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-1 mb-4 border-b border-gray-800 pb-0">
        {TABS.map((tab) => {
          const count = tab.key === "disabled"
            ? visiblePolicies.filter((p) => p.status === "disabled").length
            : visiblePolicies.filter((p) => p.policyType === "governance" && p.status !== "disabled").length;
          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`text-xs px-3 py-1.5 rounded-t border-b-2 transition-colors ${
                activeTab === tab.key
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-500 hover:text-gray-300"
              }`}
            >
              {tab.label}
              {count > 0 && (
                <span className="ml-1.5 px-1.5 py-0.5 bg-gray-800 rounded text-gray-400 text-xs">{count}</span>
              )}
            </button>
          );
        })}
      </div>

      {error && (
        <div className="bg-red-950/40 border border-red-800 rounded p-3 text-xs text-red-400 mb-4">
          Failed to load policies: {String(error)}
        </div>
      )}

      {showCreate && (
        <CreatePolicyForm
          loading={createMutation.isPending}
          error={createMutation.error ? String(createMutation.error) : null}
          onSubmit={(body) => createMutation.mutate(body)}
        />
      )}

      {isLoading && <p className="text-sm text-gray-600">Loading…</p>}

      {!isLoading && policies.length === 0 && !showCreate && (
        <div className="border border-gray-800 border-dashed rounded-lg p-12 text-center">
          <p className="text-sm text-gray-500">No policies in this view.</p>
        </div>
      )}

      <div className="space-y-3">
        {policies.map((p) => (
          <PolicyCard
            key={p.policyId}
            policy={p}
            activating={activateMutation.isPending}
            onActivate={(pvId) => activateMutation.mutate({ id: p.policyId, pvId })}
            onEdit={() => setEditPolicy(p)}
            onDisable={() => disableMutation.mutate(p.policyId)}
            onEnable={() => enableMutation.mutate(p.policyId)}
            onDelete={() => setConfirmDel(p)}
            onOpenPanel={() => openPanel("policy", p.policyId)}
          />
        ))}
      </div>
    </div>
  );
}

// ── Policy card ────────────────────────────────────────────────────────────
function PolicyCard({
  policy, activating,
  onActivate, onEdit, onDisable, onEnable, onDelete, onOpenPanel,
}: {
  policy:     PolicyListItem;
  activating: boolean;
  onActivate: (pvId: string) => void;
  onEdit:     () => void;
  onDisable:  () => void;
  onEnable:   () => void;
  onDelete:   () => void;
  onOpenPanel: () => void;
}) {
  const def    = policy.definition;
  const isTP   = policy.policyType === "third_party_monitor";
  const tpDef  = isTP ? (def as ThirdPartyMonitorDef | null) : null;
  const gDef   = !isTP ? (def as { action?: string; pattern?: string; matchTarget?: string; agentMessage?: string } | null) : null;
  const isActive = policy.status === "active" && policy.currentVersionId !== "";
  const disabled = policy.status === "disabled";

  return (
    <div className={`bg-gray-900 border rounded-lg p-4 transition-colors ${
      disabled ? "border-gray-800 opacity-60" : "border-gray-800 hover:border-gray-700"
    }`}>
      <div className="flex items-start justify-between gap-4">
        {/* Left: name + badges */}
        <div className="min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={`text-sm font-semibold ${disabled ? "line-through text-gray-500" : "text-gray-200"}`}>
              {policy.policyName}
            </span>
            <PolicyStatusBadge status={policy.status} policyType={policy.policyType} />
            {/* Governance / PII action badge */}
            {!isTP && gDef?.action && <ActionBadge action={gDef.action} />}
            {/* Third-party: their action + our action */}
            {isTP && tpDef && (
              <>
                <span className="px-2 py-0.5 rounded text-xs" style={{ background: "#fff3dc", color: "#b07400", border: "1px solid #f9ae4d", borderRadius: "3px" }}>
                  {tpDef.detected_action.toUpperCase()} detected
                </span>
                <span className="text-xs text-gray-500">→ our action:</span>
                <ActionBadge action={tpDef.our_action} />
              </>
            )}
            {isActive && !disabled && (
              <span className="flex items-center gap-1 text-xs" style={{ color: "#2d8a2d" }}>
                <CheckCircle size={11} /> v{policy.versionNumber}
              </span>
            )}
          </div>
          {policy.description && (
            <p className="text-xs text-gray-500 mt-1">{policy.description}</p>
          )}
          {!isTP && gDef?.pattern && (
            <p className="text-xs font-mono text-gray-600 mt-1">
              {gDef.matchTarget}: <span className="text-gray-400">{gDef.pattern}</span>
            </p>
          )}
          {isTP && tpDef && (
            <p className="text-xs text-gray-600 mt-1">
              Tool: <span className="font-mono" style={{ color: "#b07400" }}>{tpDef.third_party_tool}</span>
              {tpDef.occurrence_count > 0 && (
                <span className="ml-2 text-gray-600">· {tpDef.occurrence_count} occurrences</span>
              )}
            </p>
          )}
          {!isTP && gDef?.agentMessage && (
            <p className="text-xs text-gray-600 mt-1 italic">&ldquo;{gDef.agentMessage}&rdquo;</p>
          )}
        </div>

        {/* Right: action buttons */}
        <div className="flex items-center gap-1.5 shrink-0 flex-wrap justify-end">
          {/* Activate / Re-activate — only for non-disabled governance/pii */}
          {!disabled && !isTP && policy.currentVersionId && !isActive && (
            <button
              onClick={() => onActivate(policy.currentVersionId)}
              disabled={activating}
              className="text-xs px-3 py-1.5 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 text-white rounded"
            >
              Activate
            </button>
          )}

          {/* Edit */}
          <button
            onClick={onEdit}
            className="text-gray-500 hover:text-gray-200 p-1.5 rounded hover:bg-gray-800"
            title="Edit"
          >
            <Pencil size={13} />
          </button>

          {/* Disable / Enable */}
          {!disabled ? (
            <button
              onClick={onDisable}
              className="text-gray-500 hover:text-yellow-400 p-1.5 rounded hover:bg-gray-800"
              title="Disable"
            >
              <PowerOff size={13} />
            </button>
          ) : (
            <button
              onClick={onEnable}
              className="text-gray-500 hover:text-green-400 p-1.5 rounded hover:bg-gray-800"
              title="Enable"
            >
              <Power size={13} />
            </button>
          )}

          {/* Delete */}
          <button
            onClick={onDelete}
            className="text-gray-500 hover:text-red-400 p-1.5 rounded hover:bg-gray-800"
            title="Delete"
          >
            <Trash2 size={13} />
          </button>

          {/* History */}
          <button
            onClick={onOpenPanel}
            className="text-xs text-gray-500 hover:text-gray-300 px-2 py-1.5 rounded hover:bg-gray-800"
          >
            History →
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Create Policy form (unchanged from original) ──────────────────────────
const TOOL_TYPES   = ["filesystem", "shell", "mcp", "web_search", "prompt"] as const;
const ACTIONS      = ["ALERT", "BLOCK"] as const;
const MATCH_TARGETS: Record<string, string[]> = {
  filesystem: ["path"],
  shell:      ["command"],
  mcp:        ["mcp_tool"],
  web_search: ["query"],
  prompt:     ["prompt"],
};
const PATTERN_HINTS: Record<string, string> = {
  path:     "e.g.  \\.env$|\\.pem$|\\.key$",
  command:  "e.g.  rm\\s+-rf",
  mcp_tool: "e.g.  ^github/.*",
  query:    "e.g.  password|secret",
  prompt:   ".*  (matches any non-empty prompt)",
};

function CreatePolicyForm({ onSubmit, loading, error }: {
  onSubmit: (body: Parameters<typeof api.createPolicy>[0]) => void;
  loading:  boolean;
  error:    string | null;
}) {
  const [name,    setName]    = useState("");
  const [desc,    setDesc]    = useState("");
  const [type,    setType]    = useState<string>("filesystem");
  const [action,  setAction]  = useState<string>("ALERT");
  const [target,  setTarget]  = useState<string>("path");
  const [pattern, setPattern] = useState("");
  const [message, setMessage] = useState("");

  const handleTypeChange = (t: string) => {
    setType(t);
    setTarget(MATCH_TARGETS[t]?.[0] ?? "path");
    setPattern("");
  };

  const definition = {
    policyId: `pol_${Date.now()}`, policyName: name, policyVersion: 1,
    type, action, matchTarget: target, pattern,
    ...(message ? { agentMessage: message } : {}),
  };
  const canSubmit = name.trim() !== "" && pattern.trim() !== "" && !loading;
  const INPUT = "w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-600 transition-colors";

  return (
    <div className="bg-gray-900 border border-blue-800/50 rounded-lg p-5 mb-6">
      <h2 className="text-sm font-semibold text-gray-200 mb-4">New Policy</h2>
      {error && <div className="bg-red-950/40 border border-red-800 rounded p-2 text-xs text-red-400 mb-4">{error}</div>}
      <div className="grid grid-cols-2 gap-4">
        <div className="col-span-2"><label className="text-xs text-gray-500 block mb-1">Policy Name *</label>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Block credential files" className={INPUT} /></div>
        <div className="col-span-2"><label className="text-xs text-gray-500 block mb-1">Description</label>
          <input value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="Optional description" className={INPUT} /></div>
        <div><label className="text-xs text-gray-500 block mb-1">Tool Type</label>
          <select value={type} onChange={(e) => handleTypeChange(e.target.value)} className={INPUT}>
            {TOOL_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}</select></div>
        <div><label className="text-xs text-gray-500 block mb-1">Action</label>
          <select value={action} onChange={(e) => setAction(e.target.value)} className={INPUT}>
            {ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}</select></div>
        <div><label className="text-xs text-gray-500 block mb-1">Match Target</label>
          <input value={target} disabled className={INPUT + " opacity-50 cursor-not-allowed"} /></div>
        <div><label className="text-xs text-gray-500 block mb-1">Regex Pattern *</label>
          <input value={pattern} onChange={(e) => setPattern(e.target.value)} placeholder={PATTERN_HINTS[target] ?? ""} className={INPUT + " font-mono"} /></div>
        <div className="col-span-2"><label className="text-xs text-gray-500 block mb-1">Message shown to developer (optional)</label>
          <input value={message} onChange={(e) => setMessage(e.target.value)} placeholder="e.g. Credential files are restricted." className={INPUT} /></div>
      </div>
      <div className="flex items-center gap-3 mt-5">
        <button disabled={!canSubmit}
          onClick={() => onSubmit({ policyName: name.trim(), description: desc.trim(), definition, enforcementMode: action.toLowerCase(), changeSummary: "Initial version" })}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded text-xs font-medium transition-colors">
          {loading ? "Creating…" : "Create & Activate Policy"}
        </button>
      </div>
    </div>
  );
}
