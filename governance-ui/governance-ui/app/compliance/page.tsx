"use client";
// Compliance section — EU_AI_ACT_SOC2_COMPLIANCE_DESIGN.md §276:
// framework cards + per-control status, following the existing page structure.
//
// Deliberately shows raw counts rather than a "% compliant" figure. Per §5.3/§8.1
// the four statuses are not points on a scale — a partially-evidenced control is
// not half a compliant one — so averaging them would invent a number the backend
// refuses to compute.

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, type ComplianceFramework, type FrameworkControl, type ControlStatus } from "@/lib/api";
import { PageError, EmptyState } from "@/components/page-error";
import { ControlStatusBadge, CONTROL_STATUS_ORDER, controlStatusLabel } from "@/components/control-status-badge";

export default function CompliancePage() {
  const {
    data: frameworks = [],
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey:        ["compliance-frameworks"],
    queryFn:         api.complianceFrameworks,
    refetchInterval: 30000, // the backend recalculates on a 30m tick; no point polling hard
  });

  const [selected, setSelected] = useState<string | null>(null);
  // Default to the first framework once loaded, without forcing a render cycle.
  const activeId = selected ?? frameworks[0]?.frameworkId ?? null;
  const active = frameworks.find((f) => f.frameworkId === activeId) ?? null;

  return (
    <div>
      <h1 className="text-lg font-semibold text-gray-200 mb-2">Compliance</h1>
      <p className="text-xs text-gray-600 mb-6">
        Control status per framework, derived from governance evidence. Counts are reported as-is —
        no overall percentage is calculated, because a partially-evidenced control is not a fraction
        of a compliant one.
      </p>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && frameworks.length === 0 && <EmptyState label="compliance frameworks" />}

      {frameworks.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
            {frameworks.map((f) => (
              <FrameworkCard
                key={f.frameworkId}
                framework={f}
                active={f.frameworkId === activeId}
                onSelect={() => setSelected(f.frameworkId)}
              />
            ))}
          </div>

          {active && <ControlTable framework={active} />}
        </>
      )}
    </div>
  );
}

function FrameworkCard({
  framework,
  active,
  onSelect,
}: {
  framework: ComplianceFramework;
  active: boolean;
  onSelect: () => void;
}) {
  const counts: { status: ControlStatus; value: number }[] = [
    { status: "non_compliant",       value: framework.nonCompliantCount },
    { status: "partially_compliant", value: framework.partiallyCompliantCount },
    { status: "compliant",           value: framework.compliantCount },
    { status: "planned",             value: framework.plannedCount },
  ];

  return (
    <button
      onClick={onSelect}
      className={`text-left border rounded-lg p-4 transition-colors ${
        active ? "border-blue-700 bg-gray-900" : "border-gray-800 hover:bg-gray-900"
      }`}
    >
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-sm font-semibold text-gray-200">{framework.displayName}</span>
        <span className="text-xs font-mono text-gray-600">{framework.frameworkType}</span>
      </div>

      <p className="text-xs text-gray-500 mt-1">
        {framework.applicableControlCount} applicable control
        {framework.applicableControlCount === 1 ? "" : "s"}
      </p>

      <div className="grid grid-cols-4 gap-2 mt-4">
        {counts.map(({ status, value }) => (
          <div key={status}>
            <p className="text-lg font-semibold text-gray-200">{value}</p>
            <p className="text-[10px] uppercase tracking-wider text-gray-600 leading-tight">
              {controlStatusLabel(status)}
            </p>
          </div>
        ))}
      </div>

      <p className="text-[11px] text-gray-600 mt-4">
        {framework.calculatedAt
          ? `Last calculated ${new Date(framework.calculatedAt).toLocaleString()}`
          : "Not calculated yet — waiting for the first scheduler tick."}
      </p>
    </button>
  );
}

function ControlTable({ framework }: { framework: ComplianceFramework }) {
  const { data: controls = [], isLoading, isError, error } = useQuery({
    queryKey: ["framework-controls", framework.frameworkId],
    queryFn:  () => api.frameworkControls(framework.frameworkId),
  });

  // Surface problems first; within a status, keep the framework's own ordering.
  const sorted = [...controls].sort(
    (a, b) =>
      CONTROL_STATUS_ORDER.indexOf(a.currentStatus) - CONTROL_STATUS_ORDER.indexOf(b.currentStatus),
  );

  return (
    <section>
      <h2 className="text-sm font-semibold text-gray-300 mb-3">{framework.displayName} — controls</h2>

      {isError && <PageError error={error} />}
      {!isLoading && !isError && controls.length === 0 && (
        <EmptyState label={`controls mapped to ${framework.displayName}`} />
      )}

      {sorted.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Control", "Title", "Status", "Implementation", "Evidence"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sorted.map((c) => (
                <ControlRow key={c.controlId} control={c} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function ControlRow({ control }: { control: FrameworkControl }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <tr className="border-t border-gray-800 hover:bg-gray-900">
        <td className="px-4 py-3 align-top">
          <span className="font-mono font-semibold" style={{ color: "#4258c7" }}>
            {control.frameworkControlCode || control.controlCode}
          </span>
          {/* The framework's own code differs from our internal one; show both so an
              auditor can cross-reference without guessing. */}
          {control.frameworkControlCode && control.frameworkControlCode !== control.controlCode && (
            <p className="text-[10px] text-gray-600 font-mono mt-0.5">{control.controlCode}</p>
          )}
        </td>
        <td className="px-4 py-3 align-top text-gray-300">
          {control.title}
          {control.description && (
            <button
              onClick={() => setOpen((v) => !v)}
              className="ml-2 text-gray-600 hover:text-gray-400"
              aria-expanded={open}
            >
              {open ? "▾" : "▸"}
            </button>
          )}
        </td>
        <td className="px-4 py-3 align-top">
          <ControlStatusBadge status={control.currentStatus} />
        </td>
        <td className="px-4 py-3 align-top">
          <span className={control.evidenceStatus === "operational" ? "text-gray-400" : "text-gray-600"}>
            {control.evidenceStatus || "—"}
          </span>
        </td>
        <td className="px-4 py-3 align-top text-center">
          <span className={control.evidenceCount > 0 ? "text-gray-300 font-semibold" : "text-gray-600"}>
            {control.evidenceCount}
          </span>
        </td>
      </tr>
      {open && control.description && (
        <tr className="border-t border-gray-800/50 bg-gray-950">
          <td />
          <td colSpan={4} className="px-4 py-3 text-gray-500">
            {control.description}
          </td>
        </tr>
      )}
    </>
  );
}
