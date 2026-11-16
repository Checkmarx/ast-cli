// Provenance badge for third-party detection signals.
//
// Only "transcript" is a real verdict — the deciding tool's own recorded
// words. "telemetry" and "hook_correlation" are diagnostic only: they explain
// why a PostToolUse marker went missing, but neither confirms a block by
// itself (a fast shell command can start and exit between two OS-state polls,
// which looks identical to "never ran"). That's why events carrying only
// these two methods never become violations — see emitThirdPartyBlock. Colors:
// blue for a real verdict, neutral amber/gray for unconfirmed diagnostics.

const DETECTION: Record<string, { label: string; title: string; bg: string; color: string; border: string }> = {
  transcript: {
    label:  "Reported by tool",
    title:  "Read from the agent transcript — the deciding tool's own command and its own words, not an inference.",
    bg:     "#e8ebf8",
    color:  "#4258c7",
    border: "#a0ace3",
  },
  telemetry: {
    label:  "OS check: no signal",
    title:  "OS-level check found no evidence the tool ran (file unchanged / process not seen). Does not confirm a block — a fast command can start and exit between polls, which looks the same as never running.",
    bg:     "#fff3dc",
    color:  "#b07400",
    border: "#f9ae4d",
  },
  response_signature: {
    label:  "Matched from error text",
    title:  "The failed tool call's own error text matched a known security-product signature — pattern matching, not a structured verdict or a hook record. Weakest of the three ground-truth sources.",
    bg:     "#fdeee6",
    color:  "#c05621",
    border: "#e6a06b",
  },
  hook_correlation: {
    label:  "Heuristic",
    title:  "Inferred from a missing PostToolUse marker after the correlation timeout — not independently verified.",
    bg:     "#f6f6f6",
    color:  "#939499",
    border: "#ebebeb",
  },
};

/**
 * Renders nothing when the method is absent — events ingested before telemetry
 * detection existed have no provenance to report, and an empty badge would be
 * more confusing than none.
 */
export function DetectionBadge({ method }: { method?: string | null }) {
  const cfg = DETECTION[(method ?? "").toLowerCase()];
  if (!cfg) return null;
  return (
    <span
      title={cfg.title}
      className="px-2 py-0.5 text-xs font-semibold inline-block whitespace-nowrap"
      style={{
        background:   cfg.bg,
        color:        cfg.color,
        border:       `1px solid ${cfg.border}`,
        borderRadius: "3px",
        letterSpacing: "0.02em",
      }}
    >
      {cfg.label}
    </span>
  );
}
