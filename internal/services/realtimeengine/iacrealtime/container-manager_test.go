package iacrealtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// MockContainerManager for testing - does not execute real container commands
type MockContainerManager struct {
	GeneratedContainerIDs     []string
	RunKicsContainerCalls     []RunKicsContainerCall
	EnsureImageAvailableCalls []string
	ShouldFailGenerate        bool
	ShouldFailRun             bool
	ShouldFailEnsureImage     bool
	RunError                  error
	EnsureImageError          error
}

type RunKicsContainerCall struct {
	Engine    string
	VolumeMap string
}

func NewMockContainerManager() *MockContainerManager {
	return &MockContainerManager{
		GeneratedContainerIDs:     make([]string, 0),
		RunKicsContainerCalls:     make([]RunKicsContainerCall, 0),
		EnsureImageAvailableCalls: make([]string, 0),
	}
}

func (m *MockContainerManager) GenerateContainerID() string {
	if m.ShouldFailGenerate {
		return ""
	}

	containerID := uuid.New().String()
	containerName := KicsContainerPrefix + containerID
	m.GeneratedContainerIDs = append(m.GeneratedContainerIDs, containerName)
	viper.Set(commonParams.KicsContainerNameKey, containerName)
	return containerName
}

func (m *MockContainerManager) RunKicsContainer(engine, volumeMap string) error {
	call := RunKicsContainerCall{
		Engine:    engine,
		VolumeMap: volumeMap,
	}
	m.RunKicsContainerCalls = append(m.RunKicsContainerCalls, call)

	if m.ShouldFailRun {
		if m.RunError != nil {
			return m.RunError
		}
		return &exec.Error{Name: engine, Err: nil}
	}

	return nil
}

func (m *MockContainerManager) EnsureImageAvailable(engine string) error {
	m.EnsureImageAvailableCalls = append(m.EnsureImageAvailableCalls, engine)

	if m.ShouldFailEnsureImage {
		if m.EnsureImageError != nil {
			return m.EnsureImageError
		}
		return &exec.Error{Name: engine, Err: nil}
	}

	return nil
}

func TestNewMockContainerManager(t *testing.T) {
	dm := NewMockContainerManager()

	if dm == nil {
		t.Error("NewMockContainerManager() should not return nil")
		t.FailNow()
	}

	// Verify initial state
	if len(dm.GeneratedContainerIDs) != 0 {
		t.Error("New mock should have empty container IDs list")
	}

	if len(dm.RunKicsContainerCalls) != 0 {
		t.Error("New mock should have empty calls list")
	}
}

func TestMockContainerManager_GenerateContainerID(t *testing.T) {
	dm := NewMockContainerManager()

	// Clear any existing value
	viper.Set(commonParams.KicsContainerNameKey, "")

	containerName := dm.GenerateContainerID()

	// Test that a container name was generated
	if containerName == "" {
		t.Error("GenerateContainerID() should return a non-empty container name")
	}

	// Test that it has the correct prefix
	if !strings.HasPrefix(containerName, KicsContainerPrefix) {
		t.Errorf("Container name should start with prefix '%s', got '%s'", KicsContainerPrefix, containerName)
	}

	// Test that the UUID part exists (should be longer than just the prefix)
	if len(containerName) <= len(KicsContainerPrefix) {
		t.Error("Container name should include UUID after prefix")
	}

	// Test that viper was set correctly
	viperValue := viper.GetString(commonParams.KicsContainerNameKey)
	if viperValue != containerName {
		t.Errorf("Viper should be set to '%s', got '%s'", containerName, viperValue)
	}

	// Test that mock recorded the generated ID
	if len(dm.GeneratedContainerIDs) != 1 {
		t.Error("Mock should record generated container ID")
	}

	if dm.GeneratedContainerIDs[0] != containerName {
		t.Errorf("Mock should record correct container ID, got '%s', expected '%s'", dm.GeneratedContainerIDs[0], containerName)
	}

	// Test that subsequent calls generate different IDs
	containerName2 := dm.GenerateContainerID()
	if containerName == containerName2 {
		t.Error("GenerateContainerID() should generate unique container names")
	}

	// Test that mock recorded both IDs
	if len(dm.GeneratedContainerIDs) != 2 {
		t.Error("Mock should record both generated container IDs")
	}
}

func TestMockContainerManager_GenerateContainerID_Failure(t *testing.T) {
	dm := NewMockContainerManager()
	dm.ShouldFailGenerate = true

	containerName := dm.GenerateContainerID()

	if containerName != "" {
		t.Error("GenerateContainerID() should return empty string when configured to fail")
	}
}

func TestMockContainerManager_RunKicsContainer(t *testing.T) {
	dm := NewMockContainerManager()

	// Set up test parameters
	containerName := "test-container"
	viper.Set(commonParams.KicsContainerNameKey, containerName)

	tests := []struct {
		name      string
		engine    string
		volumeMap string
		expectErr bool
	}{
		{
			name:      "Valid docker engine with volume map",
			engine:    "docker",
			volumeMap: "/tmp/test:/path",
			expectErr: false,
		},
		{
			name:      "Empty engine",
			engine:    "",
			volumeMap: "/tmp/test:/path",
			expectErr: false, // Mock doesn't validate parameters
		},
		{
			name:      "Empty volume map",
			engine:    "docker",
			volumeMap: "",
			expectErr: false, // Mock doesn't validate parameters
		},
		{
			name:      "Invalid engine",
			engine:    "invalid-engine-that-does-not-exist",
			volumeMap: "/tmp/test:/path",
			expectErr: false, // Mock doesn't validate parameters
		},
		{
			name:      "FallBack engine Path verification",
			engine:    "/usr/local/bin/docker",
			volumeMap: "/tmp/test:/path",
			expectErr: false, // Mock doesn't validate parameters

		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Reset calls for this test
			dm.RunKicsContainerCalls = make([]RunKicsContainerCall, 0)

			err := dm.RunKicsContainer(ttt.engine, ttt.volumeMap)

			if ttt.expectErr && err == nil {
				t.Errorf("RunKicsContainer() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunKicsContainer() unexpected error: %v", err)
			}

			// Verify mock recorded the call
			if len(dm.RunKicsContainerCalls) != 1 {
				t.Error("Mock should record the RunKicsContainer call")
			}

			call := dm.RunKicsContainerCalls[0]
			if call.Engine != ttt.engine {
				t.Errorf("Mock should record correct engine, got '%s', expected '%s'", call.Engine, ttt.engine)
			}

			if call.VolumeMap != ttt.volumeMap {
				t.Errorf("Mock should record correct volume map, got '%s', expected '%s'", call.VolumeMap, ttt.volumeMap)
			}
		})
	}
}

func TestMockContainerManager_RunKicsContainer_Failure(t *testing.T) {
	dm := NewMockContainerManager()
	dm.ShouldFailRun = true

	err := dm.RunKicsContainer("docker", "/tmp:/path")

	if err == nil {
		t.Error("RunKicsContainer() should return error when configured to fail")
	}

	// Verify call was still recorded
	if len(dm.RunKicsContainerCalls) != 1 {
		t.Error("Mock should record the call even when configured to fail")
	}
}

func TestMockContainerManager_Integration(t *testing.T) {
	dm := NewMockContainerManager()

	// Test the full workflow
	containerName := dm.GenerateContainerID()

	// Verify container name was set in viper
	if viper.GetString(commonParams.KicsContainerNameKey) != containerName {
		t.Error("Container name should be set in viper after generation")
	}

	// Test running container
	err := dm.RunKicsContainer("docker", "/tmp:/path")
	if err != nil {
		t.Errorf("Mock RunKicsContainer should not fail by default: %v", err)
	}

	// Verify mock state
	if len(dm.GeneratedContainerIDs) != 1 {
		t.Error("Mock should record the generated container ID")
	}

	if len(dm.RunKicsContainerCalls) != 1 {
		t.Error("Mock should record the RunKicsContainer call")
	}

	call := dm.RunKicsContainerCalls[0]
	if call.Engine != "docker" || call.VolumeMap != "/tmp:/path" {
		t.Errorf("Mock should record correct call parameters: %+v", call)
	}
}

func TestMockContainerManager_EnsureImageAvailable(t *testing.T) {
	dm := NewMockContainerManager()

	err := dm.EnsureImageAvailable("docker")
	if err != nil {
		t.Errorf("EnsureImageAvailable should not fail by default: %v", err)
	}

	if len(dm.EnsureImageAvailableCalls) != 1 {
		t.Error("Mock should record EnsureImageAvailable call")
	}

	if dm.EnsureImageAvailableCalls[0] != "docker" {
		t.Errorf("Mock should record correct engine, got %s", dm.EnsureImageAvailableCalls[0])
	}
}

func TestMockContainerManager_EnsureImageAvailable_Failure(t *testing.T) {
	dm := NewMockContainerManager()
	dm.ShouldFailEnsureImage = true

	err := dm.EnsureImageAvailable("docker")
	if err == nil {
		t.Error("EnsureImageAvailable should fail when configured to fail")
	}

	// Verify call was still recorded
	if len(dm.EnsureImageAvailableCalls) != 1 {
		t.Error("Mock should record the call even when configured to fail")
	}
}

func TestCreateCommandWithEnhancedPath_ReturnsCommand(t *testing.T) {
	cmd := createCommandWithEnhancedPath("/usr/bin/docker", "run", "--rm", "hello-world")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	if cmd.Path == "" {
		t.Error("Command should have a path set")
	}

	// Verify args are passed correctly
	expectedArgs := []string{"/usr/bin/docker", "run", "--rm", "hello-world"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
	}
}

func TestCreateCommandWithEnhancedPath_NoArgs(t *testing.T) {
	cmd := createCommandWithEnhancedPath("/usr/bin/docker")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	if len(cmd.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(cmd.Args))
	}
}

func TestMockContainerManager_EnsureImageAvailable_CustomError(t *testing.T) {
	dm := NewMockContainerManager()
	dm.ShouldFailEnsureImage = true
	customErr := &exec.Error{Name: "custom", Err: nil}
	dm.EnsureImageError = customErr

	err := dm.EnsureImageAvailable("docker")
	if err != customErr {
		t.Error("EnsureImageAvailable should return custom error when set")
	}
}

// ============================================================================
// Tests for REAL ContainerManager implementation (not mock)
// ============================================================================

func TestNewContainerManager(t *testing.T) {
	cm := NewContainerManager()

	if cm == nil {
		t.Fatal("NewContainerManager() should not return nil")
	}

	// Verify it implements the interface
	var _ IContainerManager = cm
}

func TestContainerManager_GenerateContainerID(t *testing.T) {
	cm := &ContainerManager{}

	// Clear any existing value
	viper.Set(commonParams.KicsContainerNameKey, "")

	containerName := cm.GenerateContainerID()

	// Test that a container name was generated
	if containerName == "" {
		t.Error("GenerateContainerID() should return a non-empty container name")
	}

	// Test that it has the correct prefix
	if !strings.HasPrefix(containerName, KicsContainerPrefix) {
		t.Errorf("Container name should start with prefix '%s', got '%s'", KicsContainerPrefix, containerName)
	}

	// Test that the UUID part exists (should be longer than just the prefix)
	if len(containerName) <= len(KicsContainerPrefix) {
		t.Error("Container name should include UUID after prefix")
	}

	// Test that viper was set correctly
	viperValue := viper.GetString(commonParams.KicsContainerNameKey)
	if viperValue != containerName {
		t.Errorf("Viper should be set to '%s', got '%s'", containerName, viperValue)
	}

	// Test that subsequent calls generate different IDs
	containerName2 := cm.GenerateContainerID()
	if containerName == containerName2 {
		t.Error("GenerateContainerID() should generate unique container names")
	}
}

func TestContainerManager_GenerateContainerID_UUIDFormat(t *testing.T) {
	cm := &ContainerManager{}

	containerName := cm.GenerateContainerID()

	// Extract UUID part (after prefix)
	uuidPart := strings.TrimPrefix(containerName, KicsContainerPrefix)

	// UUID should be 36 characters (8-4-4-4-12 format with hyphens)
	if len(uuidPart) != 36 {
		t.Errorf("UUID part should be 36 characters, got %d: %s", len(uuidPart), uuidPart)
	}

	// Verify UUID format (contains hyphens at correct positions)
	if uuidPart[8] != '-' || uuidPart[13] != '-' || uuidPart[18] != '-' || uuidPart[23] != '-' {
		t.Errorf("UUID part should have hyphens at positions 8,13,18,23: %s", uuidPart)
	}
}

// ============================================================================
// Tests for createCommandWithEnhancedPath function
// ============================================================================

func TestCreateCommandWithEnhancedPath_ArgsPassedCorrectly(t *testing.T) {
	tests := []struct {
		name         string
		enginePath   string
		args         []string
		expectedArgs []string
	}{
		{
			name:         "Multiple args",
			enginePath:   "/usr/bin/docker",
			args:         []string{"run", "--rm", "-v", "/tmp:/data", "hello-world"},
			expectedArgs: []string{"/usr/bin/docker", "run", "--rm", "-v", "/tmp:/data", "hello-world"},
		},
		{
			name:         "Single arg",
			enginePath:   "/usr/bin/docker",
			args:         []string{"--version"},
			expectedArgs: []string{"/usr/bin/docker", "--version"},
		},
		{
			name:         "No args",
			enginePath:   "/usr/bin/podman",
			args:         []string{},
			expectedArgs: []string{"/usr/bin/podman"},
		},
		{
			name:         "Args with special characters",
			enginePath:   "/usr/local/bin/docker",
			args:         []string{"run", "--env", "VAR=value with spaces"},
			expectedArgs: []string{"/usr/local/bin/docker", "run", "--env", "VAR=value with spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createCommandWithEnhancedPath(tt.enginePath, tt.args...)

			if cmd == nil {
				t.Fatal("createCommandWithEnhancedPath should not return nil")
			}

			if len(cmd.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(cmd.Args))
			}

			for i, expected := range tt.expectedArgs {
				if i < len(cmd.Args) && cmd.Args[i] != expected {
					t.Errorf("Arg[%d]: expected %q, got %q", i, expected, cmd.Args[i])
				}
			}
		})
	}
}

func TestCreateCommandWithEnhancedPath_CommandPath(t *testing.T) {
	tests := []struct {
		name       string
		enginePath string
	}{
		{
			name:       "Absolute path",
			enginePath: "/usr/bin/docker",
		},
		{
			name:       "Relative path",
			enginePath: "./docker",
		},
		{
			name:       "Just command name",
			enginePath: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createCommandWithEnhancedPath(tt.enginePath, "--version")

			if cmd == nil {
				t.Fatal("createCommandWithEnhancedPath should not return nil")
			}

			// The first arg should always be the engine path
			if len(cmd.Args) < 1 || cmd.Args[0] != tt.enginePath {
				t.Errorf("First arg should be engine path %q, got %v", tt.enginePath, cmd.Args)
			}
		})
	}
}

func TestCreateCommandWithEnhancedPath_EnvIsSet(t *testing.T) {
	cmd := createCommandWithEnhancedPath("/usr/bin/docker", "run")

	// On non-macOS, Env might be nil (uses parent env)
	// On macOS, Env should be set with enhanced PATH
	// This test verifies the command is created without error
	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// If Env is set, verify PATH is present
	if cmd.Env != nil {
		foundPath := false
		for _, e := range cmd.Env {
			if strings.HasPrefix(e, "PATH=") {
				foundPath = true
				break
			}
		}
		if !foundPath {
			t.Error("If Env is set, it should contain PATH")
		}
	}
}

// ============================================================================
// Tests for createCommandWithEnhancedPath on macOS (mocked)
// ============================================================================

func TestCreateCommandWithEnhancedPath_MacOS_EnhancesPath(t *testing.T) {
	// Mock OS to be macOS
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	// Create a temp directory that exists (to be added to PATH)
	tempDir := t.TempDir()
	enginePath := filepath.Join(tempDir, "docker")

	cmd := createCommandWithEnhancedPath(enginePath, "run", "--rm")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// On macOS, Env should be set
	if cmd.Env == nil {
		t.Error("On macOS, cmd.Env should be set with enhanced PATH")
	}

	// Verify PATH is in the environment
	foundPath := false
	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "PATH=") {
			foundPath = true
			// Verify the engine directory is in the PATH
			if !strings.Contains(e, tempDir) {
				t.Errorf("Enhanced PATH should contain engine directory %s, got %s", tempDir, e)
			}
			break
		}
	}
	if !foundPath {
		t.Error("Enhanced PATH should contain PATH= entry")
	}
}

func TestCreateCommandWithEnhancedPath_MacOS_AddsDockerPaths(t *testing.T) {
	// Mock OS to be macOS
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	cmd := createCommandWithEnhancedPath("/usr/local/bin/docker", "--version")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// On macOS, Env should be set
	if cmd.Env == nil {
		t.Error("On macOS, cmd.Env should be set")
	}
}

func TestCreateCommandWithEnhancedPath_MacOS_DeduplicatesPaths(t *testing.T) {
	// Mock OS to be macOS
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osDarwin }

	// Set PATH to include one of the fallback paths
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()
	_ = os.Setenv("PATH", "/usr/local/bin:/usr/bin")

	cmd := createCommandWithEnhancedPath("/usr/local/bin/docker", "--version")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// Verify PATH doesn't have duplicates
	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "PATH=") {
			pathValue := strings.TrimPrefix(e, "PATH=")
			parts := strings.Split(pathValue, string(os.PathListSeparator))
			seen := make(map[string]int)
			for _, p := range parts {
				seen[p]++
				if seen[p] > 1 {
					t.Errorf("PATH contains duplicate entry: %s", p)
				}
			}
			break
		}
	}
}

func TestCreateCommandWithEnhancedPath_NonMacOS_NoEnhancement(t *testing.T) {
	// Mock OS to be Linux
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osLinux }

	cmd := createCommandWithEnhancedPath("/usr/bin/docker", "run")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// On non-macOS, Env should be nil (uses parent environment)
	if cmd.Env != nil {
		t.Error("On non-macOS, cmd.Env should be nil")
	}
}

func TestCreateCommandWithEnhancedPath_Windows_NoEnhancement(t *testing.T) {
	// Mock OS to be Windows
	origGOOS := getOS
	defer func() { getOS = origGOOS }()
	getOS = func() string { return osWindows }

	cmd := createCommandWithEnhancedPath("C:\\Program Files\\Docker\\docker.exe", "run")

	if cmd == nil {
		t.Fatal("createCommandWithEnhancedPath should not return nil")
	}

	// On Windows, Env should be nil (uses parent environment)
	if cmd.Env != nil {
		t.Error("On Windows, cmd.Env should be nil")
	}
}
