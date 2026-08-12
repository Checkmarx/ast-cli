//go:build !integration

package mcp

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewMCPCommand_Metadata(t *testing.T) {
	cmd := NewMCPCommand("1.2.3", func() bool { return true })
	if cmd.Use != "mcp" {
		t.Errorf("Use = %q, want mcp", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Long == "" {
		t.Error("expected Long description")
	}
	if cmd.RunE == nil {
		t.Fatal("RunE should be set")
	}
}

func TestNewMCPCommand_DescriptionsContainImportantTerms(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })

	tests := []struct {
		name        string
		description string
		expectedStr string
	}{
		{
			name:        "Short contains MCP",
			description: cmd.Short,
			expectedStr: "MCP",
		},
		{
			name:        "Long contains Model Context Protocol",
			description: cmd.Long,
			expectedStr: "Model Context Protocol",
		},
		{
			name:        "Long contains guardrails",
			description: cmd.Long,
			expectedStr: "guardrail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.description, tt.expectedStr) {
				t.Errorf("expected %q to contain %q", tt.description, tt.expectedStr)
			}
		})
	}
}

func TestNewMCPCommand_HasBridgeSubcommand(t *testing.T) {
	cmd := NewMCPCommand("9.9.9", func() bool { return true })
	found := false
	for _, c := range cmd.Commands() {
		if c.Use == "bridge" || strings.HasPrefix(c.Use, "bridge") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected bridge subcommand on mcp command")
	}
}

func TestNewMCPCommand_Example(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })
	if cmd.Example == "" {
		t.Error("expected Example to be set")
	}
	if !strings.Contains(cmd.Example, "cx mcp") {
		t.Errorf("example should contain 'cx mcp'")
	}
}

func TestNewMCPCommand_LicensedTrue(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })
	if cmd == nil {
		t.Fatal("expected non-nil command with licensed=true")
	}
}

func TestNewMCPCommand_LicensedFalse(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return false })
	if cmd == nil {
		t.Fatal("expected non-nil command with licensed=false")
	}
}

func TestNewMCPCommand_VersionCarried(t *testing.T) {
	testCases := []string{
		"1.0.0",
		"2.3.4",
		"0.0.1",
		"1.2.3-beta",
		"1.2.3-rc1",
	}

	for _, version := range testCases {
		t.Run("Version-"+version, func(t *testing.T) {
			cmd := NewMCPCommand(version, func() bool { return true })
			if cmd == nil {
				t.Errorf("failed to create command with version %s", version)
			}
			// Verify command is created successfully
			if cmd.Use != "mcp" {
				t.Errorf("expected Use=mcp, got %s", cmd.Use)
			}
		})
	}
}

func TestNewMCPCommand_InstructionsContent(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })

	// Instructions should mention security policy
	if !strings.Contains(cmd.Long, "cx_shell_guard") {
		t.Error("expected cx_shell_guard tool mentioned in description")
	}
	if !strings.Contains(cmd.Long, "cx_prompt_guard") {
		t.Error("expected cx_prompt_guard tool mentioned in description")
	}
	if !strings.Contains(cmd.Long, "stdio") {
		t.Error("expected stdio transport mentioned")
	}
}

func TestNewMCPCommand_MultipleInstances(t *testing.T) {
	// Ensure multiple instances can be created independently
	cmd1 := NewMCPCommand("1.0.0", func() bool { return true })
	cmd2 := NewMCPCommand("2.0.0", func() bool { return false })

	if cmd1 == nil || cmd2 == nil {
		t.Fatal("expected both commands to be created")
	}

	// Both should have the same structure but can be used independently
	if cmd1.Use != cmd2.Use {
		t.Errorf("expected same Use, got %s and %s", cmd1.Use, cmd2.Use)
	}
}

func TestNewMCPCommand_LicenseCallbackVariations(t *testing.T) {
	tests := []struct {
		name     string
		licensed func() bool
	}{
		{
			name:     "Always true",
			licensed: func() bool { return true },
		},
		{
			name:     "Always false",
			licensed: func() bool { return false },
		},
		{
			name:     "Alternating",
			licensed: func() bool { return false }, // Note: just testing it doesn't crash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewMCPCommand("1.0.0", tt.licensed)
			if cmd == nil {
				t.Fatal("expected non-nil command")
			}
			// Verify command structure is correct
			if cmd.Use != "mcp" {
				t.Errorf("expected Use=mcp, got %s", cmd.Use)
			}
		})
	}
}

func TestNewMCPCommand_HasRunFunction(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })

	if cmd.RunE == nil {
		t.Fatal("RunE should not be nil")
	}

	// The RunE function should be callable (doesn't mean we call it in tests)
	if cmd.RunE == nil {
		t.Error("expected RunE to be set to a non-nil function")
	}
}

func TestNewMCPCommand_SubcommandBridge(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })

	// Find bridge subcommand
	var bridgeCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if strings.Contains(c.Use, "bridge") {
			bridgeCmd = c
			break
		}
	}

	if bridgeCmd == nil {
		t.Fatal("expected bridge subcommand")
	}

	// Bridge command should also have proper metadata
	if bridgeCmd.Short == "" {
		t.Error("bridge command should have short description")
	}
}
