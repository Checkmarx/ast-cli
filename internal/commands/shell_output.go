package commands

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// detectShell returns the user's likely shell so session-mode login/logout
// can emit env-var assignment lines in the right syntax. PowerShell is
// detected via PSModulePath (present in PowerShell sessions, absent in
// cmd.exe and *nix shells). Bash/zsh/fish are detected via SHELL.
// Defaults: PowerShell on Windows, bash elsewhere.
func detectShell() string {
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}
	shell := strings.ToLower(os.Getenv("SHELL"))
	switch {
	case strings.Contains(shell, "fish"):
		return "fish"
	case strings.Contains(shell, "bash"), strings.Contains(shell, "zsh"):
		return "bash"
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

// formatEnvAssignment returns a shell-evaluable env var assignment line.
// Examples:
//
//	powershell  →  $env:CX_APIKEY = "value"
//	bash/zsh    →  export CX_APIKEY="value"
//	fish        →  set -gx CX_APIKEY "value"
func formatEnvAssignment(shell, name, value string) string {
	switch shell {
	case "powershell":
		return fmt.Sprintf(`$env:%s = "%s"`, name, value)
	case "fish":
		return fmt.Sprintf(`set -gx %s "%s"`, name, value)
	default:
		return fmt.Sprintf(`export %s="%s"`, name, value)
	}
}
