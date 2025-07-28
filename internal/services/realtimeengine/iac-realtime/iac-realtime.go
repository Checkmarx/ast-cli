package iac_realtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	containerPath            = "/path"
	containerFormat          = "json"
	containerTempDirPattern  = "iac-realtime"
	kicsContainerPrefix      = "cli-iac-realtime-"
	containerResultsFileName = "results.json"
)

var (
	kicsErrorCodes = []string{"60", "50", "40", "30", "20"}
)

type IacRealtimeService struct {
	JwtWrapper         wrappers.JWTWrapper
	FeatureFlagWrapper wrappers.FeatureFlagsWrapper
}

func NewIacRealtimeService(jwt wrappers.JWTWrapper, flags wrappers.FeatureFlagsWrapper) *IacRealtimeService {
	return &IacRealtimeService{
		JwtWrapper:         jwt,
		FeatureFlagWrapper: flags,
	}
}

func (svc *IacRealtimeService) RunIacRealtimeScan(filePath, ignoredFilePath string) ([]IacRealtimeResult, error) {
	if enabled, err := realtimeengine.IsFeatureFlagEnabled(svc.FeatureFlagWrapper, wrappers.OssRealtimeEnabled); err != nil || !enabled {
		logger.PrintfIfVerbose("IaC Realtime scan is not available (feature flag disabled or error: %v)", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	if err := realtimeengine.EnsureLicense(svc.JwtWrapper); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}

	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}

	svc.GenerateContainerID()

	volumeMap, tempDir, err := prepareScanEnvironment(filePath)
	defer func() {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	}()

	if err != nil {
		return nil, err
	}

	results, err := runKicsScan(volumeMap, tempDir)

	_ = os.RemoveAll(tempDir)

	return results, err
}

func prepareScanEnvironment(filePath string) (volumeMap string, tempDir string, err error) {
	if filePath == "" {
		return "", "", errorconstants.NewRealtimeEngineError("--file is required for kics-realtime command").Error()
	}

	if !hasSupportedExtension(filePath) {
		return "", "", errorconstants.NewRealtimeEngineError("Provided file is not supported by iac-realtime").Error()
	}

	tempDir, err = os.MkdirTemp("", containerTempDirPattern)
	if err != nil {
		return "", "", errorconstants.NewRealtimeEngineError("error creating temporary directory").Error()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", errorconstants.NewRealtimeEngineError("error reading file").Error()
	}

	_, file := filepath.Split(filePath)
	destPath := filepath.Join(tempDir, file)

	if err := os.WriteFile(destPath, data, os.ModePerm); err != nil {
		return "", "", errorconstants.NewRealtimeEngineError("error writing file to temporary directory").Error()
	}

	volumeMap = fmt.Sprintf("%s:%s", tempDir, containerPath)
	return volumeMap, tempDir, nil
}

func (svc *IacRealtimeService) GenerateContainerID() string {
	containerID := uuid.New().String()
	containerName := kicsContainerPrefix + containerID
	viper.Set(commonParams.KicsContainerNameKey, containerName)
	return containerName
}

func runKicsScan(volumeMap, tempDir string) ([]IacRealtimeResult, error) {
	args := []string{
		"run", "--rm",
		"-v", volumeMap,
		"--name", viper.GetString(commonParams.KicsContainerNameKey),
		util.ContainerImage,
		"scan",
		"-p", containerPath,
		"-o", containerPath,
		"--report-formats", containerFormat,
	}

	_, err := exec.Command("docker", args...).CombinedOutput()

	return handleKicsError(err, tempDir)
}

func handleKicsError(err error, tempDir string) ([]IacRealtimeResult, error) {
	msg := err.Error()
	code := extractErrorCode(msg)

	if slices.Contains(kicsErrorCodes, code) {
		results, readErr := readKicsResultsFile(tempDir)
		if readErr != nil {
			return nil, errors.Errorf("%s", readErr)
		}
		iacRealtimeResults, err := convertKicsCollectionToIacRealtimeResults(&results)
		if err != nil {
			return nil, errors.Errorf("failed to convert KICS results: %s", err)
		}
		return iacRealtimeResults, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == util.EngineNoRunningCode {
		return nil, errors.New(util.NotRunningEngineMessage)
	}

	if strings.Contains(msg, util.InvalidEngineError) || strings.Contains(msg, util.InvalidEngineErrorWindows) {
		return nil, errors.New(util.InvalidEngineMessage)
	}

	return nil, errors.Errorf("Check container engine state. Failed: %s", msg)
}

func extractErrorCode(msg string) string {
	if idx := strings.LastIndex(msg, " "); idx != -1 {
		return msg[idx+1:]
	}
	return ""
}

func readKicsResultsFile(tempDir string) (wrappers.KicsResultsCollection, error) {
	var result wrappers.KicsResultsCollection
	path := filepath.Join(tempDir, containerResultsFileName)

	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}

func hasSupportedExtension(target string) bool {
	for _, f := range commonParams.KicsBaseFilters {
		if f != "" && strings.Contains(target, f) {
			return true
		}
	}
	return false
}

func convertKicsCollectionToIacRealtimeResults(
	results *wrappers.KicsResultsCollection,
) ([]IacRealtimeResult, error) {
	var iacResults []IacRealtimeResult

	for _, result := range results.Results {
		iacResult := IacRealtimeResult{
			Title:       result.QueryName,
			Description: result.Description,
			Severity:    Severities[strings.ToLower(result.Severity)],
			FilePath:    result.Locations[0].Filename,
			Locations:   nil,
		}
		iacResults = append(iacResults, iacResult)
	}
	return iacResults, nil
}
