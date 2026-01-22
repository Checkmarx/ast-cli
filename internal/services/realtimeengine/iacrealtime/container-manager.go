package iacrealtime

import (
	"os/exec"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// IContainerManager interface for container operations
type IContainerManager interface {
	GenerateContainerID() string
	RunKicsContainer(engine, volumeMap string) error
	EnsureImageAvailable(engine string) error
}

// ContainerManager handles Docker container operations
type ContainerManager struct{}

func NewContainerManager() IContainerManager {
	return &ContainerManager{}
}

func (dm *ContainerManager) GenerateContainerID() string {
	containerID := uuid.New().String()
	containerName := KicsContainerPrefix + containerID
	viper.Set(commonParams.KicsContainerNameKey, containerName)
	return containerName
}

// EnsureImageAvailable checks if the KICS Docker image exists locally and pulls it if not available
func (dm *ContainerManager) EnsureImageAvailable(engine string) error {
	engine, err := engineNameResolution(engine, IacEnginePath)
	if err != nil {
		return err
	}

	// Check if image exists locally using 'docker image inspect'
	inspectArgs := []string{"image", "inspect", util.ContainerImage}
	if err := exec.Command(engine, inspectArgs...).Run(); err == nil {
		// Image exists locally
		return nil
	}

	// Image not found locally, attempt to pull
	logger.PrintIfVerbose("KICS Docker image not found locally. Attempting to pull: " + util.ContainerImage)

	pullArgs := []string{"pull", util.ContainerImage}
	output, pullErr := exec.Command(engine, pullArgs...).CombinedOutput()
	if pullErr != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return errors.Errorf("Failed to pull KICS Docker image '%s': %s. Please check your network connectivity or pull the image manually using: %s pull %s",
				util.ContainerImage, outputStr, engine, util.ContainerImage)
		}
		return errors.Errorf("Failed to pull KICS Docker image '%s': %v. Please check your network connectivity or pull the image manually using: %s pull %s",
			util.ContainerImage, pullErr, engine, util.ContainerImage)
	}

	logger.PrintIfVerbose("Successfully pulled KICS Docker image: " + util.ContainerImage)
	return nil
}

func (dm *ContainerManager) RunKicsContainer(engine, volumeMap string) error {
	// Ensure the KICS image is available before running
	if err := dm.EnsureImageAvailable(engine); err != nil {
		return err
	}

	engine, err := engineNameResolution(engine, IacEnginePath)
	if err != nil {
		return err
	}
	args := []string{
		"run", "--rm",
		"-v", volumeMap,
		"--name", viper.GetString(commonParams.KicsContainerNameKey),
		util.ContainerImage,
		"scan",
		"-p", ContainerPath,
		"-o", ContainerPath,
		"--report-formats", ContainerFormat,
	}
	_, err = exec.Command(engine, args...).CombinedOutput()

	return err
}

