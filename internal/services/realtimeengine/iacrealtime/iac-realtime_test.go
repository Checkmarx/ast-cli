package iacrealtime

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestFilterIgnoredFindings_WithOneIgnored(t *testing.T) {
	results := []IacRealtimeResult{
		{
			Title:        "Container Traffic Not Bound To Host Interface",
			FilePath:     "test/path/to/file.yaml",
			SimilarityID: "7540e8c3cdc3b13c3a24b8ce501d9e39fb485368e20922df18cec9564e075049",
		},
		{
			Title:        "Memory Not Limited",
			FilePath:     "test/path/to/file.yaml",
			SimilarityID: "4022c1441ba03ca00c1ad057f5e3cfb25ed165cb6b94988276bacad0485d3b74",
		},
	}

	ignored := []IgnoredIacFinding{
		{
			Title:        "Container Traffic Not Bound To Host Interface",
			SimilarityID: "7540e8c3cdc3b13c3a24b8ce501d9e39fb485368e20922df18cec9564e075049",
		},
	}

	ignoreMap := buildIgnoreMap(ignored)
	filtered := filterIgnoredFindings(results, ignoreMap)

	if len(filtered) != 1 {
		t.Fatalf("Expected 1 result after filtering, got %d", len(filtered))
	}

	if filtered[0].Title != "Memory Not Limited" {
		t.Errorf("Unexpected result after filtering: got %s, expected 'Memory Not Limited'", filtered[0].Title)
	}
}

func createExecutable(t *testing.T, tempDir, name string) string {
	t.Helper()
	path := filepath.Join(tempDir, name)
	if runtime.GOOS == "windows" {
		path += ".exe"
	}

	err := os.WriteFile(path, []byte("#!/bin/sh\necho test"), 0755)
	if err != nil {
		t.Fatalf("failed to create executable: %v", err)
	}
	return filepath.Base(path)
}

func TestEngineName_Resolution_FoundInPATH(t *testing.T) {
	tmpDir := t.TempDir()
	engineName := createExecutable(t, tmpDir, "docker")
	previousPath := os.Getenv("PATH")

	err := os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+previousPath)
	if err != nil {
		t.Fatalf("Failed to set the PATH in env")
	}
	defer func() {
		_ = os.Setenv("PATH", previousPath)
	}()
	res, err := engineNameResolution(engineName, IacEnginePath)
	if err != nil || res != engineName {
		t.Fatalf("Expected enginename in return , got %v , err %d", res, err)
	}
}

func TestEngineName_Resolution_check_fallBackPath_for_MAC_Linux(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return "darwin" } // or "linux"

	testPath := IacEnginePath
	testFile := filepath.Join(testPath, "docker")

	err := os.WriteFile(testFile, []byte("#!/bin/sh\necho test"), 0755)
	if err != nil {
		t.Skipf("skipping test, cannot write file: %v", err)
	}
	defer func() { _ = os.Remove(testFile) }()

	oldPATH := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPATH) }()
	_ = os.Setenv("PATH", "")

	result, err := engineNameResolution("docker", IacEnginePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join(IacEnginePath, "docker")
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestGetFallbackPaths_Docker_Darwin(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	fallbackDir := "/test/fallback"
	paths := getFallbackPaths(engineDocker, fallbackDir)

	// Should contain primary fallback path
	expectedPrimary := filepath.Join(fallbackDir, engineDocker)
	found := false
	for _, p := range paths {
		if p == expectedPrimary {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected primary fallback path %s in paths", expectedPrimary)
	}

	// Should contain macOS Docker fallback paths
	for _, macPath := range macOSDockerFallbackPaths {
		expectedPath := filepath.Join(macPath, engineDocker)
		if expectedPath == expectedPrimary {
			continue // Skip if same as primary
		}
		found = false
		for _, p := range paths {
			if p == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected macOS fallback path %s in paths", expectedPath)
		}
	}
}

func TestGetFallbackPaths_Podman_Darwin(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	fallbackDir := "/test/fallback"
	paths := getFallbackPaths(enginePodman, fallbackDir)

	// Should contain primary fallback path
	expectedPrimary := filepath.Join(fallbackDir, enginePodman)
	found := false
	for _, p := range paths {
		if p == expectedPrimary {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected primary fallback path %s in paths", expectedPrimary)
	}

	// Should contain macOS Podman fallback paths
	for _, macPath := range macOSPodmanFallbackPaths {
		expectedPath := filepath.Join(macPath, enginePodman)
		if expectedPath == expectedPrimary {
			continue // Skip if same as primary
		}
		found = false
		for _, p := range paths {
			if p == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected macOS fallback path %s in paths", expectedPath)
		}
	}
}

func TestGetFallbackPaths_NonDarwin(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return "linux" }

	fallbackDir := "/test/fallback"
	paths := getFallbackPaths(engineDocker, fallbackDir)

	// On non-darwin, should only contain primary fallback path
	if len(paths) != 1 {
		t.Errorf("Expected 1 path on non-darwin, got %d", len(paths))
	}

	expectedPrimary := filepath.Join(fallbackDir, engineDocker)
	if paths[0] != expectedPrimary {
		t.Errorf("Expected %s, got %s", expectedPrimary, paths[0])
	}
}

func TestGetFallbackPaths_UnknownEngine_Darwin(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	fallbackDir := "/test/fallback"
	paths := getFallbackPaths("unknown-engine", fallbackDir)

	// Should only contain primary fallback path for unknown engine
	expectedPrimary := filepath.Join(fallbackDir, "unknown-engine")
	if len(paths) != 1 {
		t.Errorf("Expected 1 path for unknown engine, got %d", len(paths))
	}
	if paths[0] != expectedPrimary {
		t.Errorf("Expected %s, got %s", expectedPrimary, paths[0])
	}
}

func TestVerifyEnginePath_NonExistentPath(t *testing.T) {
	result := verifyEnginePath("/non/existent/path/to/engine")
	if result {
		t.Error("verifyEnginePath should return false for non-existent path")
	}
}

func TestVerifyEnginePath_Directory(t *testing.T) {
	tempDir := t.TempDir()
	result := verifyEnginePath(tempDir)
	if result {
		t.Error("verifyEnginePath should return false for directory")
	}
}

func TestGetFallbackPaths_Docker_Darwin_IncludesHomePaths(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	fallbackDir := "/different/fallback"
	paths := getFallbackPaths(engineDocker, fallbackDir)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get user home directory: %v", err)
	}

	// Should contain home-based Docker paths
	expectedDockerHome := filepath.Join(homeDir, ".docker", "bin", engineDocker)
	expectedRdHome := filepath.Join(homeDir, ".rd", "bin", engineDocker)

	foundDockerHome := false
	foundRdHome := false
	for _, p := range paths {
		if p == expectedDockerHome {
			foundDockerHome = true
		}
		if p == expectedRdHome {
			foundRdHome = true
		}
	}

	if !foundDockerHome {
		t.Errorf("Expected Docker home path %s in paths", expectedDockerHome)
	}
	if !foundRdHome {
		t.Errorf("Expected Rancher Desktop home path %s in paths", expectedRdHome)
	}
}

func TestGetFallbackPaths_Podman_Darwin_IncludesHomePaths(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	fallbackDir := "/different/fallback"
	paths := getFallbackPaths(enginePodman, fallbackDir)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get user home directory: %v", err)
	}

	// Should contain home-based Podman path
	expectedPodmanHome := filepath.Join(homeDir, ".local", "bin", enginePodman)

	foundPodmanHome := false
	for _, p := range paths {
		if p == expectedPodmanHome {
			foundPodmanHome = true
		}
	}

	if !foundPodmanHome {
		t.Errorf("Expected Podman home path %s in paths", expectedPodmanHome)
	}
}

func TestGetFallbackPaths_Windows(t *testing.T) {
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osWindows }

	fallbackDir := "C:\\test\\fallback"
	paths := getFallbackPaths(engineDocker, fallbackDir)

	// On Windows, should only contain primary fallback path
	if len(paths) != 1 {
		t.Errorf("Expected 1 path on Windows, got %d", len(paths))
	}

	expectedPrimary := filepath.Join(fallbackDir, engineDocker)
	if paths[0] != expectedPrimary {
		t.Errorf("Expected %s, got %s", expectedPrimary, paths[0])
	}
}
