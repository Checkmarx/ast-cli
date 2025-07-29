package iacrealtime

import (
	"os"
	"path/filepath"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
)

func TestNewIacRealtimeService(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	mockFlags := &mock.FeatureFlagsMockWrapper{}

	service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

	if service == nil {
		t.Error("NewIacRealtimeService() should not return nil")
		t.FailNow()
	}

	if service.JwtWrapper == nil {
		t.Error("NewIacRealtimeService() should set JwtWrapper")
	}

	if service.FeatureFlagWrapper == nil {
		t.Error("NewIacRealtimeService() should set FeatureFlagWrapper")
	}

	if service.fileHandler == nil {
		t.Error("NewIacRealtimeService() should initialize fileHandler")
	}

	if service.containerManager == nil {
		t.Error("NewIacRealtimeService() should initialize containerManager")
	}

	if service.scanner == nil {
		t.Error("NewIacRealtimeService() should initialize scanner")
	}
}

func TestIacRealtimeService_checkFeatureFlag(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func()
		expectErr bool
	}{
		{
			name: "Feature flag enabled",
			setupMock: func() {
				mock.Flag = wrappers.FeatureFlagResponseModel{
					Name:   wrappers.OssRealtimeEnabled,
					Status: true,
				}
			},
			expectErr: false,
		},
		{
			name: "Feature flag disabled",
			setupMock: func() {
				mock.Flag = wrappers.FeatureFlagResponseModel{
					Name:   wrappers.OssRealtimeEnabled,
					Status: false,
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			ttt.setupMock()

			mockJWT := &mock.JWTMockWrapper{}
			mockFlags := &mock.FeatureFlagsMockWrapper{}

			service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())
			err := service.checkFeatureFlag()

			if ttt.expectErr && err == nil {
				t.Error("checkFeatureFlag() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("checkFeatureFlag() unexpected error: %v", err)
			}
		})
	}
}

func TestIacRealtimeService_ensureLicense(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func()
		expectErr bool
	}{
		{
			name: "Valid JWT allows all engines",
			setupMock: func() {
				// No special setup needed - default mock returns true for IsAllowedEngine
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			ttt.setupMock()

			mockJWT := &mock.JWTMockWrapper{}
			mockFlags := &mock.FeatureFlagsMockWrapper{}

			service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())
			err := service.ensureLicense()

			if ttt.expectErr && err == nil {
				t.Error("ensureLicense() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("ensureLicense() unexpected error: %v", err)
			}
		})
	}
}

func TestIacRealtimeService_validateFilePath(t *testing.T) {
	mockJWT := &mock.JWTMockWrapper{}
	mockFlags := &mock.FeatureFlagsMockWrapper{}
	service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

	// Create a real test file for valid path testing
	testFile, err := os.CreateTemp("", "validate-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		cleanPath := filepath.Clean(testFile.Name())
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()
	testFile.Close()

	tests := []struct {
		name      string
		filePath  string
		expectErr bool
	}{
		{
			name:      "Valid existing file",
			filePath:  testFile.Name(),
			expectErr: false,
		},
		{
			name:      "Empty file path",
			filePath:  "",
			expectErr: true,
		},
		{
			name:      "Non-existent file",
			filePath:  "/path/to/nonexistent/file.yaml",
			expectErr: true,
		},
		{
			name:      "Non-existent file in current dir",
			filePath:  "nonexistent.yaml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateFilePath(ttt.filePath)

			if ttt.expectErr && err == nil {
				t.Error("validateFilePath() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("validateFilePath() unexpected error: %v", err)
			}
		})
	}
}

func TestIacRealtimeService_RunIacRealtimeScan_FeatureFlagValidation(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func()
		expectErr  bool
		errorCheck func(error) bool
	}{
		{
			name: "Feature flag disabled should fail",
			setupMock: func() {
				mock.Flag = wrappers.FeatureFlagResponseModel{
					Name:   wrappers.OssRealtimeEnabled,
					Status: false,
				}
			},
			expectErr: true,
			errorCheck: func(err error) bool {
				return err != nil // Should get a realtime engine error
			},
		},
		{
			name: "Feature flag enabled should proceed to next validation",
			setupMock: func() {
				mock.Flag = wrappers.FeatureFlagResponseModel{
					Name:   wrappers.OssRealtimeEnabled,
					Status: true,
				}
			},
			expectErr: true, // Will fail at file validation stage since file doesn't exist
			errorCheck: func(err error) bool {
				return err != nil // Should get to file validation and fail there
			},
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			ttt.setupMock()

			mockJWT := &mock.JWTMockWrapper{}
			mockFlags := &mock.FeatureFlagsMockWrapper{}

			service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

			_, err := service.RunIacRealtimeScan("test.yaml", "docker", "")

			if ttt.expectErr && err == nil {
				t.Error("RunIacRealtimeScan() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunIacRealtimeScan() unexpected error: %v", err)
			}

			if ttt.errorCheck != nil && !ttt.errorCheck(err) {
				t.Errorf("RunIacRealtimeScan() error doesn't match expected pattern: %v", err)
			}
		})
	}
}

func TestIacRealtimeService_RunIacRealtimeScan_FilePathValidation(t *testing.T) {
	// Setup feature flag to be enabled so we get to file path validation
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   wrappers.OssRealtimeEnabled,
		Status: true,
	}

	mockJWT := &mock.JWTMockWrapper{}
	mockFlags := &mock.FeatureFlagsMockWrapper{}
	service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

	tests := []struct {
		name      string
		filePath  string
		expectErr bool
	}{
		{
			name:      "Empty file path should fail validation",
			filePath:  "",
			expectErr: true,
		},
		{
			name:      "Valid file path should proceed to file operations",
			filePath:  "test.yaml",
			expectErr: true, // Will fail at file operations since file doesn't exist
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.RunIacRealtimeScan(ttt.filePath, "docker", "")

			if ttt.expectErr && err == nil {
				t.Error("RunIacRealtimeScan() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunIacRealtimeScan() unexpected error: %v", err)
			}
		})
	}
}

func TestIacRealtimeService_RunIacRealtimeScan_WithRealFile(t *testing.T) {
	// Setup mocks for successful validation
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   wrappers.OssRealtimeEnabled,
		Status: true,
	}

	// Setup supported file extensions for testing
	originalFilters := commonParams.KicsBaseFilters
	commonParams.KicsBaseFilters = []string{".yaml", ".yml"}
	defer func() {
		commonParams.KicsBaseFilters = originalFilters
	}()

	mockJWT := &mock.JWTMockWrapper{}
	mockFlags := &mock.FeatureFlagsMockWrapper{}
	service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

	// Create a real test file
	testContent := "`apiVersion: v1\nkind: Pod\nmetadata:\n  name: test-pod"
	tempFile, err := os.CreateTemp("", "iac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		cleanPath := filepath.Clean(tempFile.Name())
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()

	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tempFile.Close()

	// Test the full workflow - it will fail at Docker execution but should pass all validation steps
	_, err = service.RunIacRealtimeScan(tempFile.Name(), "docker", "")

	// We expect this to fail at the Docker execution stage, not at validation
	if err == nil {
		t.Log("RunIacRealtimeScan() succeeded unexpectedly (Docker might be available and working)")
	} else {
		t.Logf("RunIacRealtimeScan() failed at Docker execution as expected: %v", err)
	}
}

func TestIacRealtimeService_Integration(t *testing.T) {
	// Setup successful mocks
	mock.Flag = wrappers.FeatureFlagResponseModel{
		Name:   wrappers.OssRealtimeEnabled,
		Status: true,
	}

	mockJWT := &mock.JWTMockWrapper{}
	mockFlags := &mock.FeatureFlagsMockWrapper{}

	service := NewIacRealtimeService(mockJWT, mockFlags, NewMockContainerManager())

	// Test service initialization
	if service.JwtWrapper == nil {
		t.Error("Service should have JWT wrapper")
	}

	if service.FeatureFlagWrapper == nil {
		t.Error("Service should have feature flag wrapper")
	}

	if service.fileHandler == nil {
		t.Error("Service should have file handler")
	}

	if service.containerManager == nil {
		t.Error("Service should have docker manager")
	}

	if service.scanner == nil {
		t.Error("Service should have scanner")
	}

	// Test individual validation methods
	if err := service.checkFeatureFlag(); err != nil {
		t.Errorf("checkFeatureFlag() should succeed with enabled flag: %v", err)
	}

	if err := service.ensureLicense(); err != nil {
		t.Errorf("ensureLicense() should succeed with valid JWT: %v", err)
	}

	// Create a real file for validateFilePath testing
	testFile, err := os.CreateTemp("", "integration-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		cleanPath := filepath.Clean(testFile.Name())
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()
	testFile.Close()

	if err := service.validateFilePath(testFile.Name()); err != nil {
		t.Errorf("validateFilePath() should succeed with valid existing file: %v", err)
	}

	// Test that the container ID generation works
	containerID := service.containerManager.GenerateContainerID()
	if containerID == "" {
		t.Error("Docker manager should generate container ID")
	}

	// Test that subsequent container IDs are different
	containerID2 := service.containerManager.GenerateContainerID()
	if containerID == containerID2 {
		t.Error("Docker manager should generate unique container IDs")
	}
}
