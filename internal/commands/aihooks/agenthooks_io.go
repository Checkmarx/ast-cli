package aihooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Hook input types — minimal structs capturing only the fields each handler
// needs from the agent's stdin JSON.

type toolCallHookInput struct {
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	Command   string          `json:"command,omitempty"`
	ToolInfo  struct {
		CommandLine string `json:"command_line,omitempty"`
	} `json:"tool_info,omitempty"`
}

type promptHookInput struct {
	Prompt   string `json:"prompt,omitempty"`
	ToolInfo struct {
		UserPrompt string `json:"user_prompt,omitempty"`
	} `json:"tool_info,omitempty"`
}

// readHookInput decodes one JSON object from stdin into v.
// Strips a leading UTF-8 BOM if present (Cursor on Windows prepends one).
// Uses json.Decoder so it returns as soon as a complete JSON value is read,
// without waiting for EOF — critical for Cursor, which does NOT close stdin.
func readHookInput(v any) error {
	r := bufio.NewReader(os.Stdin)
	if bom, err := r.Peek(3); err == nil && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		_, _ = r.Discard(3)
	}
	return json.NewDecoder(r).Decode(v)
}

// writeHookOutput encodes v as JSON to stdout followed by a newline.
func writeHookOutput(v any) {
	_ = json.NewEncoder(os.Stdout).Encode(v)
}

// denyViaExit2 writes a denial reason to stderr and exits with code 2.
// Windsurf and Droid use exit-code 2 to signal that the action should be blocked.
func denyViaExit2(reason string) {
	fmt.Fprintln(os.Stderr, reason)
	os.Exit(2)
}
