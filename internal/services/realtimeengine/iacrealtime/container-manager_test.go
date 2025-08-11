package iacrealtime

import (
	"os/exec"
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// MockContainerManager for testing - does not execute real container commands
type MockContainerManager struct {
	GeneratedContainerIDs []string
	RunKicsContainerCalls []RunKicsContainerCall
	ShouldFailGenerate    bool
	ShouldFailRun         bool
	RunError              error
}

type RunKicsContainerCall struct {
	Engine    string
	VolumeMap string
}

func NewMockContainerManager() *MockContainerManager {
	return &MockContainerManager{
		GeneratedContainerIDs: make([]string, 0),
		RunKicsContainerCalls: make([]RunKicsContainerCall, 0),
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
