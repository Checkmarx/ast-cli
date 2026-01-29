package iacrealtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// createCommandWithEnhancedPath creates an exec.Cmd with an enhanced PATH that includes
// Docker/Podman related directories. This is necessary on macOS when the IDE is launched
// via GUI (double-click) because it doesn't inherit the shell's PATH environment variable.
// Without this, Docker credential helpers like docker-credential-desktop won't be found.
func createCommandWithEnhancedPath(enginePath string, args ...string) *exec.Cmd {
	cmd := exec.Command(enginePath, args...)

	// Only enhance PATH on macOS
	if runtime.GOOS != "darwin" {
		return cmd
	}

	// Get current PATH
	currentPath := os.Getenv("PATH")

	// Build list of additional paths to add
	var additionalPaths []string

	// Add the directory containing the engine itself
	engineDir := filepath.Dir(enginePath)
	additionalPaths = append(additionalPaths, engineDir)

	// Add common Docker-related directories that may contain credential helpers
	additionalPaths = append(additionalPaths, macOSDockerFallbackPaths...)

	// Add user home-based paths
	if homeDir, err := os.UserHomeDir(); err == nil {
		additionalPaths = append(additionalPaths, filepath.Join(homeDir, ".docker", "bin"))
		additionalPaths = append(additionalPaths, filepath.Join(homeDir, ".rd", "bin"))
	}

	// Build a set of existing PATH entries for accurate duplicate detection
	pathParts := strings.Split(currentPath, string(os.PathListSeparator))
	pathSet := make(map[string]bool)
	for _, part := range pathParts {
		pathSet[part] = true
	}

	// Build enhanced PATH (prepend additional paths to ensure they take priority)
	var enhancedPathParts []string
	for _, p := range additionalPaths {
		// Only add if not already in PATH and directory exists
		if !pathSet[p] {
			if _, err := os.Stat(p); err == nil {
				enhancedPathParts = append(enhancedPathParts, p)
			}
		}
	}
	enhancedPathParts = append(enhancedPathParts, currentPath)
	enhancedPath := strings.Join(enhancedPathParts, string(os.PathListSeparator))

	// Set the enhanced PATH in the command's environment (replace existing PATH)
	env := os.Environ()
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + enhancedPath
			break
		}
	}
	cmd.Env = env

	logger.PrintIfVerbose("Enhanced PATH for container command: " + enhancedPath)

	return cmd
}

// EnsureImageAvailable checks if the KICS Docker image exists locally and pulls it if not available
func (dm *ContainerManager) EnsureImageAvailable(engine string) error {
	logger.PrintIfVerbose("Resolving container engine: " + engine)

	resolvedEngine, err := engineNameResolution(engine, IacEnginePath)
	if err != nil {
		logger.PrintIfVerbose("Failed to resolve container engine '" + engine + "': " + err.Error())
		return errors.Wrapf(err, "container engine '%s' not found. On macOS, if Docker is installed but not found, "+
			"try launching the IDE from terminal or ensure Docker Desktop is running", engine)
	}

	logger.PrintIfVerbose("Using container engine at: " + resolvedEngine)

	// Check if image exists locally using 'docker image inspect'
	logger.PrintIfVerbose("Checking if KICS image exists locally: " + util.ContainerImage)

	inspectCmd := createCommandWithEnhancedPath(resolvedEngine, "image", "inspect", util.ContainerImage)
	if err := inspectCmd.Run(); err == nil {
		logger.PrintIfVerbose("KICS Docker image found locally: " + util.ContainerImage)
		return nil
	}

	// Image not found locally, attempt to pull
	logger.PrintIfVerbose("KICS Docker image not found locally. Attempting to pull: " + util.ContainerImage)

	pullCmd := createCommandWithEnhancedPath(resolvedEngine, "pull", util.ContainerImage)
	output, pullErr := pullCmd.CombinedOutput()
	if pullErr != nil {
		outputStr := strings.TrimSpace(string(output))
		logger.PrintIfVerbose("Failed to pull KICS image. Output: " + outputStr)

		if outputStr != "" {
			return errors.Errorf("Failed to pull KICS Docker image '%s': %s. Please check your network connectivity or pull the image manually using: %s pull %s",
				util.ContainerImage, outputStr, resolvedEngine, util.ContainerImage)
		}
		return errors.Errorf("Failed to pull KICS Docker image '%s': %v. Please check your network connectivity or pull the image manually using: %s pull %s",
			util.ContainerImage, pullErr, resolvedEngine, util.ContainerImage)
	}

	logger.PrintIfVerbose("Successfully pulled KICS Docker image: " + util.ContainerImage)
	return nil
}

func (dm *ContainerManager) RunKicsContainer(engine, volumeMap string) error {
	// Ensure the KICS image is available before running
	if err := dm.EnsureImageAvailable(engine); err != nil {
		return err
	}

	resolvedEngine, err := engineNameResolution(engine, IacEnginePath)
	if err != nil {
		return err
	}

	cmd := createCommandWithEnhancedPath(resolvedEngine,
		"run", "--rm",
		"-v", volumeMap,
		"--name", viper.GetString(commonParams.KicsContainerNameKey),
		util.ContainerImage,
		"scan",
		"-p", ContainerPath,
		"-o", ContainerPath,
		"--report-formats", ContainerFormat,
	)
	_, err = cmd.CombinedOutput()

	return err
}
