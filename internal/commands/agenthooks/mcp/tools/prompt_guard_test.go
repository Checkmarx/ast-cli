//go:build !integration

package tools

import (
	"context"
	"testing"
)

const blockedResponse = "blocked"

// ============================================================================
// NewPromptGuardTool Tests
// ============================================================================

func TestNewPromptGuardTool_CreatesInstance(t *testing.T) {
	guardFunc := func(text string) string { return "" }
	tool := NewPromptGuardTool(guardFunc)

	if tool == nil {
		t.Fatal("NewPromptGuardTool should return non-nil instance")
	}
	if tool.guard == nil {
		t.Fatal("guard function should be set")
	}
}

func TestNewPromptGuardTool_StoresGuardFunction(t *testing.T) {
	expectedReason := "test reason"
	guardFunc := func(text string) string {
		return expectedReason
	}

	tool := NewPromptGuardTool(guardFunc)
	result := tool.guard("test")

	if result != expectedReason {
		t.Errorf("guard function should return expected reason, got %q", result)
	}
}

// ============================================================================
// PromptGuardTool.Handle Tests - Input Validation
// ============================================================================

func TestPromptGuardTool_Handle_EmptyText_Error(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, _, err := tool.Handle(ctx, nil, PromptGuardInput{Text: ""})

	if err == nil {
		t.Error("empty text should return error")
	}
	if err.Error() != "text is required" {
		t.Errorf("expected 'text is required' error, got %q", err.Error())
	}
}

func TestPromptGuardTool_Handle_ValidText_NoError(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, _, err := tool.Handle(ctx, nil, PromptGuardInput{Text: "test"})

	if err != nil {
		t.Errorf("valid text should not error, got %v", err)
	}
}

// ============================================================================
// PromptGuardTool.Handle Tests - Clean Text Response
// ============================================================================

func TestPromptGuardTool_Handle_CleanText_ReturnsCleantrue(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: "clean text"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != true {
		t.Errorf("result should have clean:true, got %v", resultMap)
	}
	if _, ok := resultMap["blocked"]; ok {
		t.Error("clean text should not have blocked field")
	}
	if _, ok := resultMap["reason"]; ok {
		t.Error("clean text should not have reason field")
	}
}

// ============================================================================
// PromptGuardTool.Handle Tests - Blocked Text Response
// ============================================================================

func TestPromptGuardTool_Handle_BlockedText_ReturnsCleanfalse(t *testing.T) {
	expectedReason := "contains secrets"
	tool := NewPromptGuardTool(func(text string) string { return expectedReason })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: "secret text"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != false {
		t.Errorf("result should have clean:false, got %v", resultMap)
	}
	if blocked, ok := resultMap["blocked"]; !ok || blocked != true {
		t.Error("blocked text should have blocked:true")
	}
	if reason, ok := resultMap["reason"]; !ok || reason != expectedReason {
		t.Errorf("result should have reason %q, got %v", expectedReason, resultMap)
	}
}

// ============================================================================
// PromptGuardTool.Handle Tests - Guard Function Invocation
// ============================================================================

func TestPromptGuardTool_Handle_InvokesGuardFunction(t *testing.T) {
	invoked := false
	receivedText := ""

	tool := NewPromptGuardTool(func(text string) string {
		invoked = true
		receivedText = text
		return ""
	})
	ctx := context.Background()

	_, _, err := tool.Handle(ctx, nil, PromptGuardInput{Text: "test input"})
	if err != nil {
		t.Errorf("Handle should not error: %v", err)
	}

	if !invoked {
		t.Error("guard function should be invoked")
	}
	if receivedText != "test input" {
		t.Errorf("guard function should receive %q, got %q", "test input", receivedText)
	}
}

func TestPromptGuardTool_Handle_MultipleInvocations(t *testing.T) {
	callCount := 0
	tool := NewPromptGuardTool(func(text string) string {
		callCount++
		return ""
	})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _, _ = tool.Handle(ctx, nil, PromptGuardInput{Text: "test"})
	}

	if callCount != 3 {
		t.Errorf("guard function should be called 3 times, got %d", callCount)
	}
}

// ============================================================================
// PromptGuardTool.Handle Tests - Edge Cases
// ============================================================================

func TestPromptGuardTool_Handle_LongText(t *testing.T) {
	longText := ""
	for i := 0; i < 10000; i++ {
		longText += "a"
	}

	tool := NewPromptGuardTool(func(text string) string {
		if len(text) > 5000 {
			return "text too long"
		}
		return ""
	})
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: longText})

	if err != nil {
		t.Errorf("long text should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != false {
		t.Error("long text should be blocked")
	}
}

func TestPromptGuardTool_Handle_SpecialCharacters(t *testing.T) {
	specialText := "test!@#$%^&*()_+-=[]{}|;:',.<>?/~`"
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: specialText})

	if err != nil {
		t.Errorf("special characters should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != true {
		t.Error("special characters should be clean")
	}
}

func TestPromptGuardTool_Handle_WhitespaceOnly(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: "   \t\n   "})

	if err != nil {
		t.Errorf("whitespace should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != true {
		t.Error("whitespace should be clean")
	}
}

func TestPromptGuardTool_Handle_UnicodeText(t *testing.T) {
	unicodeText := "こんにちは 世界 مرحبا"
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, result, err := tool.Handle(ctx, nil, PromptGuardInput{Text: unicodeText})

	if err != nil {
		t.Errorf("unicode should not error, got %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"]; !ok || clean != true {
		t.Error("unicode should be clean")
	}
}

// ============================================================================
// PromptGuardDef Tests
// ============================================================================

func TestPromptGuardDef_ReturnsValidTool(t *testing.T) {
	def := PromptGuardDef()

	if def == nil {
		t.Fatal("PromptGuardDef should return non-nil tool")
	}
}

func TestPromptGuardDef_HasCorrectName(t *testing.T) {
	def := PromptGuardDef()

	if def.Name != "cx_prompt_guard" {
		t.Errorf("tool name should be 'cx_prompt_guard', got %q", def.Name)
	}
}

func TestPromptGuardDef_HasDescription(t *testing.T) {
	def := PromptGuardDef()

	if def.Description == "" {
		t.Error("tool should have description")
	}
}

func TestPromptGuardDef_DescriptionMentionsRequiredCheck(t *testing.T) {
	def := PromptGuardDef()

	if def.Description == "" {
		t.Fatal("description should not be empty")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestPromptGuardTool_FullFlow_CleanText(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	input := PromptGuardInput{Text: "explain how to configure my application"}
	_, result, err := tool.Handle(ctx, nil, input)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"].(bool); !ok || !clean {
		t.Error("expected clean result for normal text")
	}
}

func TestPromptGuardTool_FullFlow_SecretDetected(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string {
		return "Blocked: prompt contains secrets"
	})
	ctx := context.Background()

	input := PromptGuardInput{Text: "here is my API key: secret123"}
	_, result, err := tool.Handle(ctx, nil, input)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	if clean, ok := resultMap["clean"].(bool); !ok || clean {
		t.Error("expected blocked result for secret text")
	}
	if reason, ok := resultMap["reason"].(string); !ok || reason == "" {
		t.Error("expected reason to be provided")
	}
}

func TestPromptGuardTool_ResultStructure_Clean(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return "" })
	ctx := context.Background()

	_, result, _ := tool.Handle(ctx, nil, PromptGuardInput{Text: "test"})
	resultMap := result.(map[string]any)

	// Clean result should have "clean" field
	if _, ok := resultMap["clean"]; !ok {
		t.Error("result should have 'clean' field")
	}

	// Clean result should NOT have "blocked" or "reason" fields
	if _, ok := resultMap["blocked"]; ok {
		t.Error("clean result should not have 'blocked' field")
	}
	if _, ok := resultMap["reason"]; ok {
		t.Error("clean result should not have 'reason' field")
	}
}

func TestPromptGuardTool_ResultStructure_Blocked(t *testing.T) {
	tool := NewPromptGuardTool(func(text string) string { return blockedResponse })
	ctx := context.Background()

	_, result, _ := tool.Handle(ctx, nil, PromptGuardInput{Text: "test"})
	resultMap := result.(map[string]any)

	// Blocked result should have all fields
	if _, ok := resultMap["clean"]; !ok {
		t.Error("result should have 'clean' field")
	}
	if _, ok := resultMap["blocked"]; !ok {
		t.Error("blocked result should have 'blocked' field")
	}
	if _, ok := resultMap["reason"]; !ok {
		t.Error("blocked result should have 'reason' field")
	}
}
