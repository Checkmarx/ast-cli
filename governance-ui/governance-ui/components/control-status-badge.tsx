// Control status badges, Checkmarx One palette (matching ActionBadge).
//
// The four statuses are raw counts, not a scale — "planned" means the control is
// scoped but not yet implemented, which is a different thing from failing it.
// Keeping planned visually neutral (rather than red) stops an unstarted control
// from reading as a violation.

import type { ControlStatus } from "@/lib/api";

const STATUS: Record<ControlStatus, { label: string; bg: string; color: string; border: string }> = {
  compliant:           { label: "Compliant",           bg: "#e6f9e6", color: "#2d8a2d", border: "#4dbb40" },
  partially_compliant: { label: "Partially compliant", bg: "#fff3dc", color: "#b07400", border: "#f9ae4d" },
  non_compliant:       { label: "Non-compliant",       bg: "#fde8e8", color: "#bb2a31", border: "#f1605d" },
  planned:             { label: "Planned",             bg: "#f6f6f6", color: "#939499", border: "#ebebeb" },
};

export function ControlStatusBadge({ status }: { status: ControlStatus | string }) {
  const cfg = STATUS[status as ControlStatus] ?? STATUS.planned;
  return (
    <span
      className="px-2 py-0.5 text-xs font-semibold inline-block whitespace-nowrap"
      style={{
        background:    cfg.bg,
        color:         cfg.color,
        border:        `1px solid ${cfg.border}`,
        borderRadius:  "3px",
        letterSpacing: "0.02em",
      }}
    >
      {cfg.label}
    </span>
  );
}

/** Small coloured dot for dense rows where a full badge would crowd the layout. */
export function ControlStatusDot({ status }: { status: ControlStatus | string }) {
  const cfg = STATUS[status as ControlStatus] ?? STATUS.planned;
  return <span className="inline-block w-2 h-2 rounded-full shrink-0" style={{ background: cfg.color }} />;
}

export const CONTROL_STATUS_ORDER: ControlStatus[] = [
  "non_compliant",
  "partially_compliant",
  "compliant",
  "planned",
];

export function controlStatusLabel(status: ControlStatus): string {
  return STATUS[status].label;
}
