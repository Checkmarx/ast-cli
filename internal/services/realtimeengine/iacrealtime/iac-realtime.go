package iacrealtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
)

const (
	osWindows = "windows"
)

type IacRealtimeService struct {
	JwtWrapper         wrappers.JWTWrapper
	FeatureFlagWrapper wrappers.FeatureFlagsWrapper
	fileHandler        *FileHandler
	containerManager   IContainerManager
	scanner            *Scanner
}

func NewIacRealtimeService(jwt wrappers.JWTWrapper, flags wrappers.FeatureFlagsWrapper, containerManager IContainerManager) *IacRealtimeService {
	fileHandler := NewFileHandler()
	scanner := NewScanner(containerManager)

	return &IacRealtimeService{
		JwtWrapper:         jwt,
		FeatureFlagWrapper: flags,
		fileHandler:        fileHandler,
		containerManager:   containerManager,
		scanner:            scanner,
	}
}

func loadIgnoredIacFindings(path string) ([]IgnoredIacFinding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ignored []IgnoredIacFinding
	err = json.Unmarshal(data, &ignored)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func buildIgnoreMap(ignored []IgnoredIacFinding) map[string]bool {
	m := make(map[string]bool)
	for _, f := range ignored {
		key := fmt.Sprintf("%s_%s", f.Title, f.SimilarityID)
		m[key] = true
	}
	return m
}

func filterIgnoredFindings(results []IacRealtimeResult, ignoreMap map[string]bool) []IacRealtimeResult {
	filtered := make([]IacRealtimeResult, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s_%s", r.Title, r.SimilarityID)
		if !ignoreMap[key] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (svc *IacRealtimeService) RunIacRealtimeScan(filePath, engine, ignoredFilePath string) ([]IacRealtimeResult, error) {
	err := svc.runValidations(filePath)
	if err != nil {
		return nil, err
	}

	svc.containerManager.GenerateContainerID()

	volumeMap, tempDir, err := svc.fileHandler.PrepareScanEnvironment(filePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if tempDir != "" {
			_ = svc.fileHandler.CleanupTempDirectory(tempDir)
		}
	}()

	results, err := svc.scanner.RunScan(engine, volumeMap, tempDir, filePath)
	if err != nil {
		return nil, err
	}

	if ignoredFilePath != "" {
		ignored, err := loadIgnoredIacFindings(ignoredFilePath)
		if err != nil {
			return nil, errorconstants.NewRealtimeEngineError("failed to load ignored IaC findings").Error()
		}
		ignoreMap := buildIgnoreMap(ignored)
		results = filterIgnoredFindings(results, ignoreMap)
	}

	return results, nil
}

func (svc *IacRealtimeService) runValidations(filePath string) error {
	if err := svc.checkFeatureFlag(); err != nil {
		return err
	}

	if err := svc.ensureLicense(); err != nil {
		return err
	}

	if err := svc.validateFilePath(filePath); err != nil {
		return err
	}
	return nil
}

func (svc *IacRealtimeService) checkFeatureFlag() error {
	enabled, err := realtimeengine.IsFeatureFlagEnabled(svc.FeatureFlagWrapper, wrappers.OssRealtimeEnabled)
	if err != nil || !enabled {
		return errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}
	return nil
}

func (svc *IacRealtimeService) ensureLicense() error {
	if err := realtimeengine.EnsureLicense(svc.JwtWrapper); err != nil {
		return errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}
	return nil
}

func (svc *IacRealtimeService) validateFilePath(filePath string) error {
	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		return errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}
	return nil
}

func engineNameResolution(engineName, fallBackDir string) (string, error) {
	// First, try to find the engine in PATH (works when launched from terminal)
	if _, err := exec.LookPath(engineName); err == nil {
		return engineName, nil
	}

	// On Windows, we don't have fallback paths - the engine must be in PATH
	if getOS() == osWindows {
		return "", errors.New(engineName + ": executable file not found in PATH")
	}

	// On macOS/Linux, check multiple fallback paths
	// This handles the case when IDE is launched via GUI and doesn't inherit shell PATH
	fallbackPaths := getFallbackPaths(engineName, fallBackDir)

	for _, fallbackPath := range fallbackPaths {
		if verifyEnginePath(fallbackPath) {
			return fallbackPath, nil
		}
	}

	checkedPaths := make([]string, len(fallbackPaths))
	copy(checkedPaths, fallbackPaths)
	return "", errors.Errorf("%s not found in PATH or in fallback locations: %v", engineName, checkedPaths)
}

// getFallbackPaths returns a list of paths to check for the container engine
func getFallbackPaths(engineName, fallBackDir string) []string {
	var paths []string

	// Add the primary fallback directory
	paths = append(paths, filepath.Join(fallBackDir, engineName))

	// On macOS, add additional paths based on engine type
	if getOS() == osDarwin {
		var additionalPaths []string
		switch engineName {
		case engineDocker:
			additionalPaths = macOSDockerFallbackPaths
		case enginePodman:
			additionalPaths = macOSPodmanFallbackPaths
		default:
			// Unknown engine, no additional paths
		}

		for _, dir := range additionalPaths {
			enginePath := filepath.Join(dir, engineName)
			// Avoid duplicates
			if enginePath != filepath.Join(fallBackDir, engineName) {
				paths = append(paths, enginePath)
			}
		}

		// Add user home-based paths
		if homeDir, err := os.UserHomeDir(); err == nil {
			switch engineName {
			case engineDocker:
				paths = append(paths,
					filepath.Join(homeDir, ".docker", "bin", engineDocker),
					filepath.Join(homeDir, ".rd", "bin", engineDocker))
			case enginePodman:
				paths = append(paths, filepath.Join(homeDir, ".local", "bin", enginePodman))
			default:
				// Unknown engine, no home-based paths
			}
		}
	}

	return paths
}

// verifyEnginePath checks if the engine exists and is executable at the given path
func verifyEnginePath(enginePath string) bool {
	info, err := os.Stat(enginePath)
	if err != nil || info.IsDir() {
		return false
	}

	// Verify the engine can be executed with a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), engineVerifyTimeout*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, enginePath, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

const osDarwin = "darwin"

var getOS = func() string {
	return runtime.GOOS
}
