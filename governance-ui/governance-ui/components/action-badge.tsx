// Severity badges using Checkmarx One exact colors (ast-webapp vars.less).
// BLOCK = critical red #bb2a31, ALERT = medium amber #f9ae4d, ALLOW = neutral.

const BADGE: Record<string, { bg: string; color: string; border: string }> = {
  BLOCK: { bg: "#fde8e8", color: "#bb2a31", border: "#f1605d" },
  ALERT: { bg: "#fff3dc", color: "#b07400", border: "#f9ae4d" },
  ALLOW: { bg: "#e8ebf8", color: "#4258c7", border: "#a0ace3" },
};

export function ActionBadge({ action }: { action: string }) {
  const key  = (action ?? "").toUpperCase();
  const cfg  = BADGE[key] ?? { bg: "#f6f6f6", color: "#939499", border: "#ebebeb" };
  return (
    <span
      className="px-2 py-0.5 rounded text-xs font-semibold inline-block"
      style={{
        background:  cfg.bg,
        color:       cfg.color,
        border:      `1px solid ${cfg.border}`,
        borderRadius: "3px",
        letterSpacing: "0.02em",
      }}
    >
      {key || "—"}
    </span>
  );
}

export function SeverityBadge({ severity }: { severity: string }) {
  const map: Record<string, { bg: string; color: string }> = {
    critical: { bg: "#fde8e8", color: "#bb2a31" },
    high:     { bg: "#fde8e8", color: "#f1605d" },
    medium:   { bg: "#fff3dc", color: "#b07400" },
    low:      { bg: "#f6f6f6", color: "#939499" },
    info:     { bg: "#e8ebf8", color: "#4258c7" },
  };
  const cfg = map[severity?.toLowerCase()] ?? { bg: "#f6f6f6", color: "#939499" };
  return (
    <span
      className="px-2 py-0.5 rounded text-xs font-semibold capitalize"
      style={{ background: cfg.bg, color: cfg.color, border: `1px solid ${cfg.color}33`, borderRadius: "3px" }}
    >
      {severity || "—"}
    </span>
  );
}

export function SeverityDot({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    critical: "#bb2a31",
    high:     "#f1605d",
    medium:   "#f9ae4d",
    low:      "#bfbdc2",
    info:     "#4258c7",
  };
  const color = colors[severity?.toLowerCase()] ?? "#bfbdc2";
  return (
    <span
      className="inline-block w-2 h-2 rounded-full shrink-0"
      style={{ background: color }}
    />
  );
}
