//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/containersrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/stretchr/testify/assert"
)

func TestContainersRealtimeScan_PositiveDockerfile_Success(t *testing.T) {
	//t.Skip("Skipping this test till the RT api for containers will deploy to DEU ENV")
	configuration.LoadConfiguration()
	args := []string{
		"scan", "containers-realtime",
		"-s", "data/positive/Dockerfile",
	}
	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "Sending positive Dockerfile should not fail")

	var images containersrealtime.ContainerImageResults
	err = json.Unmarshal(bytes.Bytes(), &images)
	assert.Nil(t, err, "Failed to unmarshal container image results")

	assert.Equal(t, 1, len(images.Images), "Should return exactly one image")

	if len(images.Images) == 0 {
		t.Fatal("No images found in the scan results")
	}
	img := images.Images[0]
	assert.Equal(t, "rabbitmq", img.ImageName, "Image name should match")
	assert.Equal(t, "4.2", img.ImageTag, "Image tag should match")
	assert.Equal(t, "Critical", img.Status, "Image status should be 'Critical'")
	assert.GreaterOrEqual(t, len(img.Vulnerabilities), 61, "Should have at least 61 vulnerabilities")
}

func TestContainersRealtimeScan_WithSeverityThreshold_Success(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "containers-realtime",
		"-s", "data/positive/Dockerfile",
		"--severity-threshold", "Critical",
	}
	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "Sending positive Dockerfile should not fail")
	var images containersrealtime.ContainerImageResults
	err = json.Unmarshal(bytes.Bytes(), &images)
	assert.Nil(t, err, "Failed to unmarshal container image results")
	assert.Equal(t, 1, len(images.Images), "Should return exactly one image")
	if len(images.Images) == 0 {
		t.Fatal("No images found in the scan results")
	}
	img := images.Images[0]
	assert.Equal(t, "rabbitmq", img.ImageName, "Image name should match")
	assert.Equal(t, "4.2", img.ImageTag, "Image tag should match")
	assert.Equal(t, "Critical", img.Status, "Image status should be 'Critical'")

	for _, vulnerability := range img.Vulnerabilities {
		assert.Equal(t, "Critical", vulnerability.Severity, "All vulnerabilities should be 'Critical'")
	}
}

func TestContainersRealtimeScan_EmptyDockerfile_SuccessWithEmptyResponse(t *testing.T) {
	configuration.LoadConfiguration()
	args := []string{
		"scan", "containers-realtime",
		"-s", "data/empty/Dockerfile",
	}
	err, bytes := executeCommand(t, args...)
	assert.Nil(t, err, "Sending empty Dockerfile should not fail")

	var images containersrealtime.ContainerImageResults
	err = json.Unmarshal(bytes.Bytes(), &images)
	assert.Nil(t, err, "Failed to unmarshal container image results")
	assert.Equal(t, len(images.Images), 0, "Should return no images")
	assert.NotNil(t, images.Images)
}
