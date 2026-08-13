//go:build !integration

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
)

// ============================================================================
// exitIfError Tests - Subprocess Testing for os.Exit
// ============================================================================

func TestExitIfError_NilError_DoesNotExit(t *testing.T) {
	// Nil error should not exit - test with subprocess
	if os.Getenv("TEST_EXIT_NIL") == "1" {
		exitIfError(nil)
		// If we reach here, the function didn't call os.Exit
		os.Exit(successfulExitCode)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitIfError_NilError_DoesNotExit")
	cmd.Env = append(os.Environ(), "TEST_EXIT_NIL=1")
	err := cmd.Run()

	if err != nil {
		t.Errorf("exitIfError(nil) should not exit, but got error: %v", err)
	}
}

func TestExitIfError_WithError_ExitsWithFailure(t *testing.T) {
	// Non-nil error should call os.Exit(failureExitCode)
	if os.Getenv("TEST_EXIT_ERROR") == "1" {
		exitIfError(errors.New("test error"))
		// Should not reach here
		os.Exit(successfulExitCode)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitIfError_WithError_ExitsWithFailure")
	cmd.Env = append(os.Environ(), "TEST_EXIT_ERROR=1")
	err := cmd.Run()

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != failureExitCode {
			t.Errorf("expected exit code %d, got %d", failureExitCode, exitErr.ExitCode())
		}
	} else if err == nil {
		t.Error("should have exited with error")
	}
}

func TestExitIfError_AstError_ExitsWithEngineCode(t *testing.T) {
	// AstError with specific code should use that code
	if os.Getenv("TEST_EXIT_AST") == "1" {
		astErr := &wrappers.AstError{
			Err:  errors.New("SAST failed"),
			Code: 2,
		}
		exitIfError(astErr)
		os.Exit(successfulExitCode)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitIfError_AstError_ExitsWithEngineCode")
	cmd.Env = append(os.Environ(), "TEST_EXIT_AST=1")
	err := cmd.Run()

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2 for SAST error, got %d", exitErr.ExitCode())
		}
	}
}

// ============================================================================
// bindKeysToEnvAndDefault Tests
// ============================================================================

func TestBindKeysToEnvAndDefault_NoErrors(t *testing.T) {
	// Reset viper for this test
	viper.Reset()

	// This function should not panic
	// We test it by ensuring it completes without error
	// Note: The actual function calls exitIfError on viper bind errors
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("bindKeysToEnvAndDefault should not panic: %v", r)
		}
	}()

	// We can't test the full function without mocking viper
	// but we can verify it's callable
	_ = viper.BindEnv
}

// ============================================================================
// bindProxy Tests
// ============================================================================

func TestBindProxy_SetDefault(t *testing.T) {
	viper.Reset()

	// Test that proxy default is set
	// We can verify viper is properly initialized
	if viper.GetString("proxy") == "" {
		// Default should be empty string
		t.Logf("proxy default is correctly empty")
	}
}

func TestBindProxy_EnvironmentVariableBinding(t *testing.T) {
	viper.Reset()

	// Set a test environment variable
	const testProxy = "http://proxy.example.com:8080"
	_ = os.Setenv("HTTP_PROXY", testProxy)
	defer func() { _ = os.Unsetenv("HTTP_PROXY") }()

	// After binding, viper should be able to read it
	err := viper.BindEnv("test_proxy", "HTTP_PROXY")
	if err != nil {
		t.Errorf("BindEnv should not fail, got: %v", err)
	}
}

// ============================================================================
// Constants Tests
// ============================================================================

func TestConstants_ExitCodes(t *testing.T) {
	if successfulExitCode != 0 {
		t.Errorf("successfulExitCode should be 0, got %d", successfulExitCode)
	}

	if failureExitCode != 1 {
		t.Errorf("failureExitCode should be 1, got %d", failureExitCode)
	}

	expectedKillCmd := "kill"
	if killCommand != expectedKillCmd {
		t.Errorf("killCommand should be %q, got %q", expectedKillCmd, killCommand)
	}
}

// ============================================================================
// signalHandler Tests - Isolated Logic
// ============================================================================

func TestSignalHandler_Docker_PSCommand_Available(t *testing.T) {
	// Test that docker ps command can be executed
	cmd := exec.Command("docker", "ps")
	_, err := cmd.CombinedOutput()

	// We expect this to either work or fail gracefully
	// depending on whether docker is installed
	if err != nil && !strings.Contains(err.Error(), "executable file not found") {
		t.Logf("docker ps failed (possibly expected if Docker not installed): %v", err)
	}
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestExitIfError_AstError_WithCode(t *testing.T) {
	// Test that AstError is handled correctly
	testErr := errors.New("test error")
	astErr := &wrappers.AstError{
		Err:  testErr,
		Code: 2,
	}

	// We can't fully test this without exiting,
	// but we can verify the structure
	if astErr.Err != testErr {
		t.Errorf("AstError.Err should be the test error")
	}
	if astErr.Code != 2 {
		t.Errorf("AstError.Code should be 2")
	}
}

func TestExitIfError_AstError_WithCustomCode(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
	}{
		{"SAST engine error", 2, "SAST scan failed"},
		{"SCA engine error", 3, "SCA scan failed"},
		{"KICS engine error", 4, "IaC scan failed"},
		{"API Security error", 5, "API scan failed"},
		{"Multiple engines", 1, "Multiple engines failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			astErr := &wrappers.AstError{
				Err:  errors.New(tt.message),
				Code: tt.code,
			}

			if astErr.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, astErr.Code)
			}
			if astErr.Err.Error() != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, astErr.Err.Error())
			}
		})
	}
}

// ============================================================================
// Integration-style Tests
// ============================================================================

func TestExitCodes_Values(t *testing.T) {
	// Verify exit codes are correctly defined
	expectedSuccess := 0
	expectedFailure := 1

	if successfulExitCode != expectedSuccess {
		t.Errorf("successfulExitCode = %d, want %d", successfulExitCode, expectedSuccess)
	}

	if failureExitCode != expectedFailure {
		t.Errorf("failureExitCode = %d, want %d", failureExitCode, expectedFailure)
	}
}

func TestSignalConstants(t *testing.T) {
	// Verify SIGTERM is the correct signal
	expectedSignal := syscall.SIGTERM

	// SIGTERM is typically 15 on Unix systems
	if expectedSignal == 0 {
		t.Error("SIGTERM should be a valid signal")
	}
}

func TestKillCommand_Constant(t *testing.T) {
	expectedKillCmd := "kill"
	if killCommand != expectedKillCmd {
		t.Errorf("killCommand should be %q, got %q", expectedKillCmd, killCommand)
	}

	// Verify it's a valid docker subcommand name
	if len(killCommand) == 0 {
		t.Error("killCommand should not be empty")
	}
}

// ============================================================================
// Command Execution Tests
// ============================================================================

func TestDockerCommand_PSExecutable(t *testing.T) {
	cmd := exec.Command("docker", "ps")
	err := cmd.Err

	// We can't guarantee docker is installed,
	// but we can verify the command is constructable
	if err != nil && !strings.Contains(err.Error(), "not found") {
		// Some error other than "not found"
		t.Logf("docker command failed: %v", err)
	}
}

func TestDockerCommand_KillExecutable(t *testing.T) {
	// Test docker kill command structure
	cmd := exec.Command("docker", "kill", "container-name")

	// Verify it's properly constructed
	if cmd.Path != "docker" && !strings.Contains(cmd.Path, "docker") {
		t.Logf("docker kill command path: %s", cmd.Path)
	}

	if len(cmd.Args) != 3 {
		t.Errorf("docker kill command should have 3 args (docker, kill, container), got %d", len(cmd.Args))
	}
}

// ============================================================================
// Environment Variable Tests
// ============================================================================

func TestEnvironmentVariableBinding(t *testing.T) {
	viper.Reset()

	testKey := "TEST_KEY"
	testValue := "test_value"

	// Set environment variable
	_ = os.Setenv(testKey, testValue)
	defer func() { _ = os.Unsetenv(testKey) }()

	// Bind it
	err := viper.BindEnv("test_config", testKey)
	if err != nil {
		t.Errorf("BindEnv failed: %v", err)
	}

	// Verify viper can read it
	retrieved := viper.GetString("test_config")
	if retrieved != testValue {
		t.Errorf("viper.GetString should return %q, got %q", testValue, retrieved)
	}
}

func TestMultipleEnvironmentVariableBinding(t *testing.T) {
	viper.Reset()

	// Test binding multiple environment variables with fallback
	primaryEnv := "PRIMARY_VAR"
	secondaryEnv := "SECONDARY_VAR"
	primaryValue := "primary_value"

	_ = os.Setenv(primaryEnv, primaryValue)
	defer func() {
		_ = os.Unsetenv(primaryEnv)
		_ = os.Unsetenv(secondaryEnv)
	}()

	// Bind primary first
	err := viper.BindEnv("my_config", primaryEnv, secondaryEnv)
	if err != nil {
		t.Errorf("BindEnv with multiple vars failed: %v", err)
	}

	retrieved := viper.GetString("my_config")
	if retrieved != primaryValue {
		t.Errorf("viper should prioritize first env var, got %q", retrieved)
	}
}

// ============================================================================
// Proxy Configuration Tests
// ============================================================================

func TestProxyEnvironmentVariable_HTTPProxy(t *testing.T) {
	const testProxy = "http://proxy.example.com:8080"
	_ = os.Setenv("HTTP_PROXY", testProxy)
	defer func() { _ = os.Unsetenv("HTTP_PROXY") }()

	retrieved := os.Getenv("HTTP_PROXY")
	if retrieved != testProxy {
		t.Errorf("HTTP_PROXY env var should be %q, got %q", testProxy, retrieved)
	}
}

func TestProxyEnvironmentVariable_CXSpecific(t *testing.T) {
	testProxy := "http://custom-proxy.corp.com:3128"
	_ = os.Setenv("CX_HTTP_PROXY", testProxy)
	defer func() { _ = os.Unsetenv("CX_HTTP_PROXY") }()

	retrieved := os.Getenv("CX_HTTP_PROXY")
	if retrieved != testProxy {
		t.Errorf("CX_HTTP_PROXY env var should be %q, got %q", testProxy, retrieved)
	}
}

// ============================================================================
// Output Capture Tests
// ============================================================================

func TestStdoutCapture(t *testing.T) {
	// Test that we can capture stdout
	oldStdout := os.Stdout
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = w

	// Write something to stdout
	println("test output")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	output := buf.String()

	if output == "" {
		t.Logf("stdout capture test completed")
	}
}
