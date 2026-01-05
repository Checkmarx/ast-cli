package iacrealtime

import (
	"github.com/google/uuid"
	"os/exec"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

// IContainerManager interface for container operations
type IContainerManager interface {
	GenerateContainerID() string
	RunKicsContainer(engine, volumeMap string) error
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

func (dm *ContainerManager) RunKicsContainer(engine, volumeMap string) error {
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

	var msg string
	if err != nil {
		msg = err.Error()
		if strings.Contains(msg, util.InvalidEngineError) {
			enginePath, err := checkEnginePresentInPath(engine)
			if err != nil {
				_, err = exec.Command(enginePath, args...).CombinedOutput()
			}
		}
	}
	return err
}
