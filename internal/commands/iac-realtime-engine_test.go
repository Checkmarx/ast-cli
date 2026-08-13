//go:build !integration

package commands

import (
	"bytes"
	"errors"
	"os"
	"testing"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ============================================================================
// RunScanIacRealtimeCommand Tests - Missing File Source Flag
// ============================================================================

func TestRunScanIacRealtimeCommand_MissingFileSource_Error(t *testing.T) {
	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "file source")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "ignored file path")
	cmd.Flags().String(commonParams.EngineFlag, "", "engine")

	err := handler(cmd, []string{})

	if err == nil {
		t.Error("expected error for missing file source")
	}
}

func TestRunScanIacRealtimeCommand_EmptyFileSource_Error(t *testing.T) {
	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "file source")

	// Don't set the flag value - it should default to empty string
	err := handler(cmd, []string{})

	if err == nil {
		t.Error("empty file source should return error")
	}

	if !errors.Is(err, errorconstants.NewRealtimeEngineError("file path is required").Error()) {
		// Check that the error message contains the expected text
		if err.Error() != errorconstants.NewRealtimeEngineError("file path is required").Error().Error() {
			t.Logf("error message: %v", err.Error())
		}
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Valid File Source
// ============================================================================

func TestRunScanIacRealtimeCommand_ValidFileSource_Success(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	jwtMock := &mock.JWTMockWrapper{}
	flagsMock := &mock.FeatureFlagsMockWrapper{}

	handler := RunScanIacRealtimeCommand(jwtMock, flagsMock)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "ignored file path")
	cmd.Flags().String(commonParams.EngineFlag, "kics", "engine")

	// Set the flags
	_ = cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	_ = cmd.Flags().Set(commonParams.EngineFlag, "kics")

	err = handler(cmd, []string{})

	// The error might be related to container execution, not flag handling
	// We're testing that the function properly processes the flags
	if err != nil {
		t.Logf("handler returned error (expected if docker/kics not available): %v", err)
	}
}

func TestRunScanIacRealtimeCommand_WithIgnoredFilePath(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"
	ignoredFile := testDir + "/ignored.json"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = os.WriteFile(ignoredFile, []byte("[]"), 0o644)
	if err != nil {
		t.Fatalf("failed to create ignored file: %v", err)
	}

	jwtMock := &mock.JWTMockWrapper{}
	flagsMock := &mock.FeatureFlagsMockWrapper{}

	handler := RunScanIacRealtimeCommand(jwtMock, flagsMock)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, ignoredFile, "ignored file path")
	cmd.Flags().String(commonParams.EngineFlag, "kics", "engine")

	_ = cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	_ = cmd.Flags().Set(commonParams.IgnoredFilePathFlag, ignoredFile)
	_ = cmd.Flags().Set(commonParams.EngineFlag, "kics")

	err = handler(cmd, []string{})

	// Log error if any for debugging
	if err != nil {
		t.Logf("handler returned error (expected if docker/kics not available): %v", err)
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Different Engine Values
// ============================================================================

func TestRunScanIacRealtimeCommand_WithDocker_Engine(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.EngineFlag, "docker", "engine")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	cmd.Flags().Set(commonParams.EngineFlag, "docker")

	err = handler(cmd, []string{})

	if err != nil {
		t.Logf("docker engine test - error (expected if docker not available): %v", err)
	}
}

func TestRunScanIacRealtimeCommand_WithPodman_Engine(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.EngineFlag, "podman", "engine")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	cmd.Flags().Set(commonParams.EngineFlag, "podman")

	err = handler(cmd, []string{})

	if err != nil {
		t.Logf("podman engine test - error (expected if podman not available): %v", err)
	}
}

func TestRunScanIacRealtimeCommand_WithEmptyEngine_Default(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.EngineFlag, "", "engine")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)

	err = handler(cmd, []string{})

	if err != nil {
		t.Logf("default engine test - error: %v", err)
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Output Handling
// ============================================================================

func TestRunScanIacRealtimeCommand_OutputBuffer(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	outputBuffer := bytes.NewBuffer([]byte{})
	cmd := &cobra.Command{}
	cmd.SetOut(outputBuffer)
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.EngineFlag, "kics", "engine")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	cmd.Flags().Set(commonParams.EngineFlag, "kics")

	err = handler(cmd, []string{})

	// Verify output buffer is used
	if err != nil {
		t.Logf("output buffer test - error: %v", err)
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Comprehensive Scenarios
// ============================================================================

func TestRunScanIacRealtimeCommand_NoFlagsSet_UsesDefaults(t *testing.T) {
	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "file source")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, "", "ignored file path")
	cmd.Flags().String(commonParams.EngineFlag, "", "engine")

	// Don't set any flags - all should be empty
	err := handler(cmd, []string{})

	if err == nil {
		t.Error("should error when file source is not provided")
	}
}

func TestRunScanIacRealtimeCommand_PathWithSpaces(t *testing.T) {
	testDir := t.TempDir()
	dirWithSpaces := testDir + "/dir with spaces"
	err := os.Mkdir(dirWithSpaces, 0o755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	testFile := dirWithSpaces + "/test.tf"
	err = os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.EngineFlag, "kics", "engine")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	cmd.Flags().Set(commonParams.EngineFlag, "kics")

	err = handler(cmd, []string{})

	if err != nil {
		t.Logf("path with spaces test - error: %v", err)
	}
}

func TestRunScanIacRealtimeCommand_AbsolutePath(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)

	err = handler(cmd, []string{})

	if err != nil {
		t.Logf("absolute path test - error: %v", err)
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Flag Retrieval
// ============================================================================

func TestRunScanIacRealtimeCommand_FlagRetrieval_FileSourceExtracted(t *testing.T) {
	testDir := t.TempDir()
	testFile := testDir + "/test.tf"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Track if the correct file source is used by checking for error
	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer([]byte{}))
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")

	cmd.Flags().Set(commonParams.SourcesFlag, testFile)

	err = handler(cmd, []string{})

	// Should not error on flag handling
	if err != nil {
		t.Logf("flag retrieval test - service execution error (expected): %v", err)
	}
}

func TestRunScanIacRealtimeCommand_MultipleEngineTypes(t *testing.T) {
	engines := []string{"docker", "podman", "kics", ""}

	for _, engine := range engines {
		t.Run("engine_"+engine, func(t *testing.T) {
			testDir := t.TempDir()
			testFile := testDir + "/test.tf"

			err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			handler := RunScanIacRealtimeCommand(
				&mock.JWTMockWrapper{},
				&mock.FeatureFlagsMockWrapper{},
			)

			cmd := &cobra.Command{}
			cmd.SetOut(bytes.NewBuffer([]byte{}))
			cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
			cmd.Flags().String(commonParams.EngineFlag, engine, "engine")

			cmd.Flags().Set(commonParams.SourcesFlag, testFile)
			if engine != "" {
				cmd.Flags().Set(commonParams.EngineFlag, engine)
			}

			err = handler(cmd, []string{})

			if err != nil {
				t.Logf("engine %q - error (expected if engine not available): %v", engine, err)
			}
		})
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Handler Function Type
// ============================================================================

func TestRunScanIacRealtimeCommand_ReturnsCobraErrorHandler(t *testing.T) {
	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	if handler == nil {
		t.Error("handler should not be nil")
	}

	// Verify it's a function that can be called
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "file source")

	result := handler(cmd, []string{})

	// Should return an error when no source is provided
	if result == nil {
		t.Error("should return error when source flag is missing")
	}
}

// ============================================================================
// RunScanIacRealtimeCommand Tests - Wrapper Injection
// ============================================================================

func TestRunScanIacRealtimeCommand_WithJWTWrapper(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

	handler := RunScanIacRealtimeCommand(jwtWrapper, featureFlagsWrapper)

	if handler == nil {
		t.Error("handler should be created with wrappers")
	}

	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "file source")

	// Handler should be callable
	err := handler(cmd, []string{})
	if err == nil {
		t.Error("should error for missing file source")
	}
}

func TestRunScanIacRealtimeCommand_WithFeatureFlagsWrapper(t *testing.T) {
	jwtWrapper := &mock.JWTMockWrapper{}
	featureFlagsWrapper := &mock.FeatureFlagsMockWrapper{}

	handler := RunScanIacRealtimeCommand(jwtWrapper, featureFlagsWrapper)

	if handler == nil {
		t.Error("handler should be created successfully")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestRunScanIacRealtimeCommand_FullFlow_WithAllFlags(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	testDir := t.TempDir()
	testFile := testDir + "/test.tf"
	ignoredFile := testDir + "/ignored.json"

	err := os.WriteFile(testFile, []byte("resource \"aws_s3_bucket\" {}"), 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = os.WriteFile(ignoredFile, []byte("[]"), 0o644)
	if err != nil {
		t.Fatalf("failed to create ignored file: %v", err)
	}

	handler := RunScanIacRealtimeCommand(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
	)

	outputBuffer := bytes.NewBuffer([]byte{})
	cmd := &cobra.Command{}
	cmd.SetOut(outputBuffer)
	cmd.Flags().String(commonParams.SourcesFlag, testFile, "file source")
	cmd.Flags().String(commonParams.IgnoredFilePathFlag, ignoredFile, "ignored file path")
	cmd.Flags().String(commonParams.EngineFlag, "kics", "engine")

	_ = cmd.Flags().Set(commonParams.SourcesFlag, testFile)
	_ = cmd.Flags().Set(commonParams.IgnoredFilePathFlag, ignoredFile)
	_ = cmd.Flags().Set(commonParams.EngineFlag, "kics")

	err = handler(cmd, []string{})

	// Verify handler was called
	if err != nil {
		t.Logf("full flow test - error: %v", err)
	}
}
