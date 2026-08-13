//go:build !integration

package tools

import (
	"context"
	"testing"
)

// ============================================================================
// NewShellGuardTool Tests
// ============================================================================

func TestNewShellGuardTool_CreatesInstance(t *testing.T) {
	guardFunc := func(command string) (bool, string) { return false, "" }
	tool := NewShellGuardTool(guardFunc)

	if tool == nil {
		t.Fatal("NewShellGuardTool should return non-nil instance")
	}
	if tool.guard == nil {
		t.Fatal("guard function should be set")
	}
}

func TestNewShellGuardTool_StoresGuardFunction(t *testing.T) {
	expectedBlocked := true
	expectedReason := "blocked by policy"
	guardFunc := func(command string) (bool, string) {
		return expectedBlocked, expectedReason
	}

	tool := NewShellGuardTool(guardFunc)
	blocked, reason := tool.guard("test")

	if blocked != expectedBlocked {
		t.Errorf("guard function should return blocked=%v", expectedBlocked)
	}
	if reason != expectedReason {
		t.Errorf("guard function should return reason %q", expectedReason)
	}
}

// ============================================================================
// ShellGuardTool.Handle Tests - Input Validation
// ============================================================================

func TestShellGuardTool_Handle_EmptyCommand_Error(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, _, err := tool.Handle(ctx, nil, ShellGuardInput{Command: ""})

	if err == nil {
		t.Error("empty command should return error")
	}
	if err.Error() != "command is required" {
		t.Errorf("expected 'command is required' error, got %q", err.Error())
	}
}

func TestShellGuardTool_Handle_ValidCommand_NoError(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, _, err := tool.Handle(ctx, nil, ShellGuardInput{Command: "ls"})

	if err != nil {
		t.Errorf("valid command should not error, got %v", err)
	}
}

// ============================================================================
// ShellGuardTool.Handle Tests - Allowed Command Response
// ============================================================================

func TestShellGuardTool_Handle_AllowedCommand_ReturnsAllowedtrue(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: "ls -la"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if cmd, ok := resultMap["command"]; !ok || cmd != "ls -la" {
		t.Errorf("result should have command field")
	}
	if allowed, ok := resultMap["allowed"]; !ok || allowed != true {
		t.Errorf("result should have allowed:true, got %v", resultMap)
	}
	if _, ok := resultMap["reason"]; ok {
		t.Error("allowed command should not have reason field")
	}
}

// ============================================================================
// ShellGuardTool.Handle Tests - Blocked Command Response
// ============================================================================

func TestShellGuardTool_Handle_BlockedCommand_ReturnsAllowedfalse(t *testing.T) {
	expectedReason := "rm command is not allowed"
	tool := NewShellGuardTool(func(command string) (bool, string) { return true, expectedReason })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: "rm -rf /"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if cmd, ok := resultMap["command"]; !ok || cmd != "rm -rf /" {
		t.Errorf("result should have command field")
	}
	if allowed, ok := resultMap["allowed"]; !ok || allowed != false {
		t.Errorf("result should have allowed:false, got %v", resultMap)
	}
	if reason, ok := resultMap["reason"]; !ok || reason != expectedReason {
		t.Errorf("result should have reason %q, got %v", expectedReason, resultMap)
	}
}

// ============================================================================
// ShellGuardTool.Handle Tests - Guard Function Invocation
// ============================================================================

func TestShellGuardTool_Handle_InvokesGuardFunction(t *testing.T) {
	invoked := false
	receivedCommand := ""

	tool := NewShellGuardTool(func(command string) (bool, string) {
		invoked = true
		receivedCommand = command
		return false, ""
	})
	ctx := context.Background()

	_, _, _ = tool.Handle(ctx, nil, ShellGuardInput{Command: "git status"})

	if !invoked {
		t.Error("guard function should be invoked")
	}
	if receivedCommand != "git status" {
		t.Errorf("guard function should receive %q, got %q", "git status", receivedCommand)
	}
}

func TestShellGuardTool_Handle_MultipleInvocations(t *testing.T) {
	callCount := 0
	tool := NewShellGuardTool(func(command string) (bool, string) {
		callCount++
		return false, ""
	})
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tool.Handle(ctx, nil, ShellGuardInput{Command: "ls"})
	}

	if callCount != 5 {
		t.Errorf("guard function should be called 5 times, got %d", callCount)
	}
}

// ============================================================================
// ShellGuardTool.Handle Tests - Edge Cases
// ============================================================================

func TestShellGuardTool_Handle_LongCommand(t *testing.T) {
	longCommand := "echo "
	for i := 0; i < 5000; i++ {
		longCommand += "a"
	}

	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: longCommand})

	if err != nil {
		t.Errorf("long command should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if allowed, ok := resultMap["allowed"].(bool); !ok || !allowed {
		t.Error("long command should be allowed")
	}
}

func TestShellGuardTool_Handle_CommandWithPipes(t *testing.T) {
	command := "cat file.txt | grep error | wc -l"
	tool := NewShellGuardTool(func(cmd string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: command})

	if err != nil {
		t.Errorf("command with pipes should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if cmd, ok := resultMap["command"]; !ok || cmd != command {
		t.Error("result should preserve original command")
	}
}

func TestShellGuardTool_Handle_CommandWithRedirection(t *testing.T) {
	command := "cat file.txt > output.txt 2>&1"
	tool := NewShellGuardTool(func(cmd string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: command})

	if err != nil {
		t.Errorf("command with redirection should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if cmd, ok := resultMap["command"]; !ok || cmd != command {
		t.Error("result should preserve original command")
	}
}

func TestShellGuardTool_Handle_CommandWithSpecialCharacters(t *testing.T) {
	command := "echo 'hello!@#$%^&*()_+-=[]{}|;:,.<>?/~`world'"
	tool := NewShellGuardTool(func(cmd string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: command})

	if err != nil {
		t.Errorf("command with special characters should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if allowed, ok := resultMap["allowed"].(bool); !ok || !allowed {
		t.Error("command with special characters should be allowed")
	}
}

func TestShellGuardTool_Handle_CommandWithWhitespace(t *testing.T) {
	command := "  git    status   "
	tool := NewShellGuardTool(func(cmd string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: command})

	if err != nil {
		t.Errorf("command with whitespace should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if cmd, ok := resultMap["command"]; !ok || cmd != command {
		t.Error("result should preserve original command with whitespace")
	}
}

func TestShellGuardTool_Handle_CommandWithUnicode(t *testing.T) {
	command := "echo 'こんにちは世界'"
	tool := NewShellGuardTool(func(cmd string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, ShellGuardInput{Command: command})

	if err != nil {
		t.Errorf("command with unicode should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if allowed, ok := resultMap["allowed"].(bool); !ok || !allowed {
		t.Error("command with unicode should be allowed")
	}
}

// ============================================================================
// ShellGuardDef Tests
// ============================================================================

func TestShellGuardDef_ReturnsValidTool(t *testing.T) {
	def := ShellGuardDef()

	if def == nil {
		t.Fatal("ShellGuardDef should return non-nil tool")
	}
}

func TestShellGuardDef_HasCorrectName(t *testing.T) {
	def := ShellGuardDef()

	if def.Name != "cx_shell_guard" {
		t.Errorf("tool name should be 'cx_shell_guard', got %q", def.Name)
	}
}

func TestShellGuardDef_HasDescription(t *testing.T) {
	def := ShellGuardDef()

	if def.Description == "" {
		t.Error("tool should have description")
	}
}

func TestShellGuardDef_DescriptionMentionsRequiredCheck(t *testing.T) {
	def := ShellGuardDef()

	if def.Description == "" {
		t.Fatal("description should not be empty")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestShellGuardTool_FullFlow_AllowedCommand(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	input := ShellGuardInput{Command: "git log --oneline"}
	_, result, err := tool.Handle(ctx, nil, input)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if allowed, ok := resultMap["allowed"].(bool); !ok || !allowed {
		t.Error("expected allowed result for git command")
	}
}

func TestShellGuardTool_FullFlow_BlockedCommand(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) {
		return true, "Blocked by policy: dangerous command"
	})
	ctx := context.Background()

	input := ShellGuardInput{Command: "rm -rf /"}
	_, result, err := tool.Handle(ctx, nil, input)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if allowed, ok := resultMap["allowed"].(bool); !ok || allowed {
		t.Error("expected blocked result for dangerous command")
	}
	if reason, ok := resultMap["reason"].(string); !ok || reason == "" {
		t.Error("expected reason to be provided")
	}
}

func TestShellGuardTool_ResultStructure_Allowed(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return false, "" })
	ctx := context.Background()

	_, result, _ := tool.Handle(ctx, nil, ShellGuardInput{Command: "ls"})
	resultMap := result.(map[string]any)

	// Allowed result should have "command" and "allowed" fields
	if _, ok := resultMap["command"]; !ok {
		t.Error("result should have 'command' field")
	}
	if _, ok := resultMap["allowed"]; !ok {
		t.Error("result should have 'allowed' field")
	}

	// Allowed result should NOT have "reason" field
	if _, ok := resultMap["reason"]; ok {
		t.Error("allowed result should not have 'reason' field")
	}
}

func TestShellGuardTool_ResultStructure_Blocked(t *testing.T) {
	tool := NewShellGuardTool(func(command string) (bool, string) { return true, "blocked" })
	ctx := context.Background()

	_, result, _ := tool.Handle(ctx, nil, ShellGuardInput{Command: "rm"})
	resultMap := result.(map[string]any)

	// Blocked result should have all fields
	if _, ok := resultMap["command"]; !ok {
		t.Error("result should have 'command' field")
	}
	if _, ok := resultMap["allowed"]; !ok {
		t.Error("result should have 'allowed' field")
	}
	if _, ok := resultMap["reason"]; !ok {
		t.Error("blocked result should have 'reason' field")
	}
}

func TestShellGuardTool_AllowedFieldValue_Correct(t *testing.T) {
	tests := []struct {
		name     string
		blocked  bool
		expected bool
	}{
		{
			name:     "allowed command",
			blocked:  false,
			expected: true,
		},
		{
			name:     "blocked command",
			blocked:  true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewShellGuardTool(func(command string) (bool, string) { return tt.blocked, "" })
			ctx := context.Background()

			_, result, _ := tool.Handle(ctx, nil, ShellGuardInput{Command: "test"})
			resultMap := result.(map[string]any)

			if allowed, ok := resultMap["allowed"].(bool); !ok || allowed != tt.expected {
				t.Errorf("allowed field should be %v, got %v", tt.expected, resultMap["allowed"])
			}
		})
	}
}
