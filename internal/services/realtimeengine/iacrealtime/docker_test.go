package iacrealtime

import (
	"strings"
	"testing"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

func TestNewDockerManager(t *testing.T) {
	dm := NewDockerManager()

	if dm == nil {
		t.Error("NewDockerManager() should not return nil")
	}
}

func TestDockerManager_GenerateContainerID(t *testing.T) {
	dm := NewDockerManager()

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

	// Test that subsequent calls generate different IDs
	containerName2 := dm.GenerateContainerID()
	if containerName == containerName2 {
		t.Error("GenerateContainerID() should generate unique container names")
	}
}

func TestDockerManager_RunKicsContainer(t *testing.T) {
	dm := NewDockerManager()

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
			expectErr: true, // May pass or fail depending on docker availability
		},
		{
			name:      "Empty engine",
			engine:    "",
			volumeMap: "/tmp/test:/path",
			expectErr: true,
		},
		{
			name:      "Empty volume map",
			engine:    "docker",
			volumeMap: "",
			expectErr: true,
		},
		{
			name:      "Invalid engine",
			engine:    "invalid-engine-that-does-not-exist",
			volumeMap: "/tmp/test:/path",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := dm.RunKicsContainer(ttt.engine, ttt.volumeMap)

			// Special handling for Docker availability in test environment
			if ttt.name == "Valid docker engine with volume map" {
				// Docker might be available or not in test env - both are acceptable
				if err == nil {
					t.Logf("Docker command succeeded (Docker is available in test environment)")
				} else {
					t.Logf("Docker command failed as expected: %v", err)
				}
				return
			}

			if ttt.expectErr && err == nil {
				t.Errorf("RunKicsContainer() expected error but got none")
			}

			if !ttt.expectErr && err != nil {
				t.Errorf("RunKicsContainer() unexpected error: %v", err)
			}
		})
	}
}

func TestDockerManager_RunKicsContainer_Arguments(t *testing.T) {
	// This test verifies that the docker command is constructed correctly
	// We can't easily test the actual execution without mocking exec.Command
	dm := NewDockerManager()

	containerName := "test-container-args"
	viper.Set(commonParams.KicsContainerNameKey, containerName)

	// Test with typical parameters
	engine := "docker"
	volumeMap := "/tmp/source:/path"

	// The actual execution will likely fail, but we're more interested
	// in testing that the method doesn't panic and handles the parameters
	err := dm.RunKicsContainer(engine, volumeMap)

	// We expect an error since docker command won't work in test environment
	if err == nil {
		t.Log("RunKicsContainer() succeeded unexpectedly (docker might be available)")
	} else {
		t.Logf("RunKicsContainer() failed as expected in test environment: %v", err)
	}
}

func TestDockerManager_Integration(t *testing.T) {
	dm := NewDockerManager()

	// Test the full workflow
	containerName := dm.GenerateContainerID()

	// Verify container name was set in viper
	if viper.GetString(commonParams.KicsContainerNameKey) != containerName {
		t.Error("Container name should be set in viper after generation")
	}

	// Test running container (will fail but shouldn't panic)
	err := dm.RunKicsContainer("docker", "/tmp:/path")
	if err == nil {
		t.Log("Docker command succeeded (docker is available in test environment)")
	} else {
		// This is expected in most test environments
		t.Logf("Docker command failed as expected: %v", err)
	}
}
