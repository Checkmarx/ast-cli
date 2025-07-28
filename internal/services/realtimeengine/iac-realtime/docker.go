package iac_realtime

import (
	"os/exec"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// DockerManager handles Docker container operations
type DockerManager struct{}

func NewDockerManager() *DockerManager {
	return &DockerManager{}
}

func (dm *DockerManager) GenerateContainerID() string {
	containerID := uuid.New().String()
	containerName := KicsContainerPrefix + containerID
	viper.Set(commonParams.KicsContainerNameKey, containerName)
	return containerName
}

func (dm *DockerManager) RunKicsContainer(engine, volumeMap string) error {
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

	_, err := exec.Command(engine, args...).CombinedOutput()
	return err
}
