package aihooks

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
)

type hookCategory int

const (
	hookIdle hookCategory = iota
	hookToolCall
	hookFileWrite
	hookPrompt
)

// Route classification and dispatch
var routeCategories = map[string]hookCategory{
	"claude-stop":                    hookIdle,
	"cursor-stop":                    hookIdle,
	"windsurf-post-cascade-response": hookIdle,
	"droid-stop":                     hookIdle,
	"gemini-after-agent":             hookIdle,

	"claude-pre-tool-use":       hookToolCall,
	"cursor-before-shell":       hookToolCall,
	"cursor-before-mcp":         hookToolCall,
	"windsurf-pre-run-command":  hookToolCall,
	"windsurf-pre-mcp-tool-use": hookToolCall,
	"droid-pre-tool-use":        hookToolCall,
	"gemini-before-tool":        hookToolCall,

	"claude-after-file-write":  hookFileWrite,
	"cursor-after-file-edit":   hookFileWrite,
	"windsurf-post-write-code": hookFileWrite,
	"droid-after-file-write":   hookFileWrite,
	"gemini-after-file-tool":   hookFileWrite,

	"claude-user-prompt-submit":   hookPrompt,
	"cursor-before-submit-prompt": hookPrompt,
	"windsurf-pre-user-prompt":    hookPrompt,
	"droid-user-prompt-submit":    hookPrompt,
	"gemini-before-agent":         hookPrompt,
}

// NewAIHooksCommand returns the top-level "aihooks" command with all 22
// hidden route subcommands. Agents invoke: cx aihooks <route-name>
func NewAIHooksCommand() *cobra.Command {
	aiHooksCmd := &cobra.Command{
		Use:   "ai-hooks",
		Short: "AI coding agent hooks for guardrail enforcement",
		Long:  "Guardrail hooks for AI coding agents (Claude, Cursor, Windsurf, Factory Droid, Gemini). Each subcommand reads JSON from stdin and writes a verdict as JSON to stdout.",
		Example: heredoc.Doc(`
			$ cx aihooks claude-pre-tool-use   < input.json
			$ cx aihooks cursor-before-shell   < input.json
		`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.com/resource/documents/en/34965-365503-hooks.html
			`),
		},
	}

	type route struct{ use, short string }

	routes := []route{
		{"claude-stop", "Claude agent finished"},
		{"cursor-stop", "Cursor agent finished"},
		{"windsurf-post-cascade-response", "Windsurf agent finished"},
		{"droid-stop", "Factory Droid agent finished"},
		{"gemini-after-agent", "Gemini agent finished"},

		{"claude-pre-tool-use", "Gate Claude tool execution"},
		{"cursor-before-shell", "Gate Cursor shell execution"},
		{"cursor-before-mcp", "Gate Cursor MCP execution"},
		{"windsurf-pre-run-command", "Gate Windsurf shell execution"},
		{"windsurf-pre-mcp-tool-use", "Gate Windsurf MCP execution"},
		{"droid-pre-tool-use", "Gate Droid tool execution"},
		{"gemini-before-tool", "Gate Gemini tool execution"},

		{"claude-after-file-write", "React to Claude file write"},
		{"cursor-after-file-edit", "React to Cursor file edit"},
		{"windsurf-post-write-code", "React to Windsurf file write"},
		{"droid-after-file-write", "React to Droid file write"},
		{"gemini-after-file-tool", "React to Gemini file write"},

		{"claude-user-prompt-submit", "Gate Claude prompt"},
		{"cursor-before-submit-prompt", "Gate Cursor prompt"},
		{"windsurf-pre-user-prompt", "Gate Windsurf prompt"},
		{"droid-user-prompt-submit", "Gate Droid prompt"},
		{"gemini-before-agent", "Gate Gemini prompt"},
	}

	for _, r := range routes {
		r := r
		aiHooksCmd.AddCommand(&cobra.Command{
			Use:               r.use,
			Short:             r.short,
			Hidden:            true,
			PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
			Run: func(cmd *cobra.Command, _ []string) {
				runHook(cmd.Use, isLicensed())
			},
		})
	}

	return aiHooksCmd
}

func runHook(route string, licensed bool) {
	category, ok := routeCategories[route]
	if !ok {
		fmt.Fprintf(os.Stderr, "ai-hooks: unknown route %q\n", route)
		os.Exit(1)
	}

	switch category {
	case hookIdle:
		handleIdleHook(route)
	case hookToolCall:
		handleToolCallHook(route, licensed)
	case hookFileWrite:
		handleFileWriteHook()
	case hookPrompt:
		handlePromptHook(route, licensed)
	}
}

func isLicensed() bool {
	_ = configuration.LoadConfiguration()
	jwt := wrappers.NewJwtWrapper()
	for _, engine := range []string{
		params.CheckmarxOneAssistType,
		params.AIProtectionType,
		params.CheckmarxDevAssistType,
	} {
		if ok, err := jwt.IsAllowedEngine(engine); err == nil && ok {
			return true
		}
	}
	return false
}

// No-op handlers — idle and file-write events have no guardrail today.
func handleIdleHook(route string) {
	var raw json.RawMessage
	_ = readHookInput(&raw)

	switch route {
	case "claude-stop", "droid-stop":
		writeHookOutput(map[string]any{"continue": true})
	default:
		writeHookOutput(struct{}{})
	}
}

func handleFileWriteHook() {
	var raw json.RawMessage
	_ = readHookInput(&raw)
	writeHookOutput(struct{}{})
}
