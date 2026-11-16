-- Migration 016: record how Claude Code was launched (CLAUDE_CODE_ENTRYPOINT).
-- ide_name alone is ambiguous: `claude` run in VS Code's integrated terminal and the
-- VS Code extension both report ide_name = 'vscode'. Entrypoint ('cli', 'claude-vscode',
-- 'claude-desktop', 'sdk-py', ...) distinguishes them, and lets the UI say a session
-- was started from a terminal.

ALTER TABLE ai_sessions
    ADD COLUMN IF NOT EXISTS ide_entrypoint VARCHAR(50);
