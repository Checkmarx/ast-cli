// Where a session was started from. Shared by the sessions list and the session panel.
//
// name       = host application (vscode, cursor, terminal, ...)
// entrypoint = how Claude Code was launched ("cli" = the `claude` CLI, "claude-vscode" = the
//              extension). `claude` run in an IDE's integrated terminal has name "vscode" but
//              entrypoint "cli", so it is labelled "VS Code · Terminal", not the extension.

const HOST_META: Record<string, { label: string; bg: string; dot: string }> = {
  vscode:    { label: "VS Code",    bg: "#4258c7", dot: "#a0ace3" },
  jetbrains: { label: "JetBrains",  bg: "#9747ff", dot: "#c9a6ff" },
  cursor:    { label: "Cursor",     bg: "#1a1a1a", dot: "#888888" },
  windsurf:  { label: "Windsurf",   bg: "#0e7490", dot: "#67d7f0" },
  desktop:   { label: "Desktop",    bg: "#565360", dot: "#bfbdc2" },
  sdk:       { label: "Agent SDK",  bg: "#565360", dot: "#bfbdc2" },
  terminal:  { label: "Terminal",   bg: "#2b2a33", dot: "#4dbb40" },
};

export interface IDEInfo {
  name?:          string;
  version?:       string;
  workspacePath?: string;
  entrypoint?:    string;
}

export function ideLabel(ide?: IDEInfo): string {
  if (!ide?.name) return "Unknown";
  const host = HOST_META[ide.name]?.label ?? ide.name;
  const fromTerminal = ide.entrypoint === "cli" && ide.name !== "terminal";
  return fromTerminal ? `${host} · Terminal` : host;
}

export function IDEBadge({ ide, showVersion = false }: { ide?: IDEInfo; showVersion?: boolean }) {
  // No IDE reported at all: usually a session that began before the hooks were installed.
  if (!ide?.name) {
    return (
      <span title="No session-start event received, so the launch environment is unknown"
            style={{ color: "#939499", fontSize: "11px", border: "1px dashed #bfbdc2", borderRadius: 4, padding: "2px 7px" }}>
        Unknown
      </span>
    );
  }
  const meta = HOST_META[ide.name] ?? { label: ide.name, bg: "#565360", dot: "#bfbdc2" };
  const fromTerminal = ide.entrypoint === "cli" && ide.name !== "terminal";
  return (
    <span
      title={fromTerminal ? `${meta.label} integrated terminal (claude CLI)` : ide.entrypoint ? `Entrypoint: ${ide.entrypoint}` : undefined}
      style={{
        display: "inline-flex", alignItems: "center", gap: "5px",
        background: meta.bg, color: "#ffffff", borderRadius: "4px",
        padding: "3px 8px", fontSize: "11px", fontWeight: 600,
        letterSpacing: "0.02em", whiteSpace: "nowrap",
      }}
    >
      <span style={{ width: 6, height: 6, borderRadius: "50%", background: meta.dot, flexShrink: 0 }} />
      {meta.label}
      {fromTerminal && <span style={{ opacity: 0.8, fontFamily: "monospace" }}>&gt;_</span>}
      {showVersion && ide.version ? <span style={{ opacity: 0.75 }}>{ide.version}</span> : null}
    </span>
  );
}
