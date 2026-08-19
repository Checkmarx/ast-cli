// Package cursorplugin holds Cursor-plugin-specific fragments for agent-hook remediation
// guidance (MCP tool names, PowerShell stop-parsing suppress commands).
package cursorplugin

import (
	"fmt"
	"runtime"
	"strings"
)

// MCPServerID is how Cursor names the Checkmarx MCP when cx-devassist is installed as a plugin
// (plugin id "cx-devassist" + mcp.json server key "Checkmarx").
const MCPServerID = "plugin-cx-devassist-Checkmarx"

// MCPTool returns the fully-qualified Cursor MCP tool name for a remediation tool.
func MCPTool(tool string) string {
	return "mcp__" + MCPServerID + "__" + tool
}

const goosWindows = "windows"

// IgnoreVulnerabilityCommand renders `cx ignore-vulnerability` for Cursor agents.
// On Windows, uses PowerShell --% with the JSON wrapped in double quotes and inner quotes
// backslash-escaped — the only form that survives PowerShell's native argv parsing.
func IgnoreVulnerabilityCommand(cxBinary, scanType string, data []byte, ignoreFlag, provenance string) string {
	if runtime.GOOS == goosWindows {
		escaped := escapeJSONForStopParsing(string(data))
		// escaped is already quote-escaped for PowerShell's double-quoted string rules;
		// %q would re-escape it using Go's own rules (e.g. doubling backslashes) and corrupt it.
		return fmt.Sprintf(`  & %q --%% ignore-vulnerability --scan-type %s --data "%s"%s%s`, //nolint:gocritic
			cxBinary, scanType, escaped, ignoreFlag, provenance)
	}
	escaped := escapeJSONForPOSIX(string(data))
	// escaped is already quote-escaped for the POSIX shell's double-quoted string rules;
	// %q would re-escape it using Go's own rules (e.g. doubling backslashes) and corrupt it.
	return fmt.Sprintf(`  %s ignore-vulnerability --scan-type %s --data "%s"%s%s`, //nolint:gocritic
		cxBinary, scanType, escaped, ignoreFlag, provenance)
}

func escapeJSONForStopParsing(data string) string {
	return strings.ReplaceAll(data, `"`, `\"`)
}

func escapeJSONForPOSIX(data string) string {
	return strings.ReplaceAll(data, `"`, `\"`)
}
