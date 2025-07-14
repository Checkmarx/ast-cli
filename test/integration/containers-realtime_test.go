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

	img := images.Images[0]
	assert.Equal(t, "confluentinc/cp-kafkacat", img.ImageName, "Image name should match")
	assert.Equal(t, "6.1.10", img.ImageTag, "Image tag should match")
	assert.Equal(t, "Malicious", img.Status, "Image status should be 'Malicious'")
	assert.GreaterOrEqual(t, len(img.Vulnerabilities), 110, "Should have at least 110 vulnerabilities")
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
