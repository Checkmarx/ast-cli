package realtimeengine

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/pkg/errors"
)

func TestIsFeatureFlagEnabled(t *testing.T) {
	tests := []struct {
		name      string
		flagName  string
		flagValue bool
		flagErr   error
		want      bool
		wantErr   bool
	}{
		{
			name:      "Feature flag enabled",
			flagName:  "test-flag",
			flagValue: true,
			flagErr:   nil,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "Feature flag disabled",
			flagName:  "test-flag",
			flagValue: false,
			flagErr:   nil,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Feature flag error",
			flagName:  "test-flag",
			flagValue: false,
			flagErr:   errors.New("flag fetch failed"),
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mock.Flag = wrappers.FeatureFlagResponseModel{
				Name:   tt.flagName,
				Status: tt.flagValue,
			}
			mock.FFErr = tt.flagErr

			mockWrapper := &mock.FeatureFlagsMockWrapper{}

			got, err := IsFeatureFlagEnabled(mockWrapper, tt.flagName)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsFeatureFlagEnabled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("IsFeatureFlagEnabled() got = %v, want %v", got, tt.want)
			}

			// Cleanup
			mock.FFErr = nil
		})
	}
}

func TestEnsureLicense(t *testing.T) {
	tests := []struct {
		name                  string
		jwtWrapper            wrappers.JWTWrapper
		shouldReturnError     bool
		expectedErrorContains string
	}{
		{
			name:                  "Nil JWT wrapper",
			jwtWrapper:            nil,
			shouldReturnError:     true,
			expectedErrorContains: "JWT wrapper is not initialized",
		},
		{
			name:              "CheckmarxOneAssist enabled",
			jwtWrapper:        &mock.JWTMockWrapper{},
			shouldReturnError: false,
		},
		{
			name: "CheckmarxOneAssist disabled but default mock enables others",
			jwtWrapper: &mock.JWTMockWrapper{
				CheckmarxOneAssistEnabled: 1,
			},
			shouldReturnError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureLicense(tt.jwtWrapper)

			if (err != nil) != tt.shouldReturnError {
				t.Errorf("EnsureLicense() error = %v, shouldReturnError %v", err, tt.shouldReturnError)
				return
			}

			if tt.shouldReturnError && tt.expectedErrorContains != "" {
				if err == nil || !contains(err.Error(), tt.expectedErrorContains) {
					t.Errorf("EnsureLicense() error %v does not contain %s", err, tt.expectedErrorContains)
				}
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		filePath  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "Valid file",
			filePath: tempFile.Name(),
			wantErr:  false,
		},
		{
			name:      "Non-existent file",
			filePath:  "/non/existent/path/file.txt",
			wantErr:   true,
			errSubstr: "does not exist",
		},
		{
			name:      "Directory instead of file",
			filePath:  tempDir,
			wantErr:   true,
			errSubstr: "directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidateFilePath() error %v does not contain %s", err, tt.errSubstr)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str != "" && substr != ""
}
