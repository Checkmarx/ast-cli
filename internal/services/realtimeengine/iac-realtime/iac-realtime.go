package iac_realtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	containerImage           = "checkmarx/kics:v2.1.11"
	containerPath            = "/path"
	containerFormat          = "json"
	containerTempDirPattern  = "kics"
	kicsContainerPrefix      = "cli-kics-realtime-"
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

func (svc *IacRealtimeService) RunIacRealtimeScan(filePath, ignoredFilePath string) (*IacRealtimeResults, error) {
	containerID := uuid.New().String()
	containerName := kicsContainerPrefix + containerID
	viper.Set(commonParams.KicsContainerNameKey, containerName)

	volumeMap, tempDir, err := prepareScanEnvironment(filePath)
	if err != nil {
		return nil, err
	}

	err = runKicsScan(volumeMap, tempDir)
	logger.PrintIfVerbose("Removing folder in temp")
	removeErr := os.RemoveAll(tempDir)
	if removeErr != nil {
		logger.PrintIfVerbose(removeErr.Error())
	}

	return nil, err
}

func prepareScanEnvironment(filePath string) (string, string, error) {
	if filePath == "" {
		return "", "", errors.New("--file is required for kics-realtime command")
	}

	if !contains(commonParams.KicsBaseFilters, filePath) {
		return "", "", errors.Errorf("%s. Provided file is not supported by kics", filePath)
	}

	tempDir, err := os.MkdirTemp("", containerTempDirPattern)
	if err != nil {
		return "", "", errors.New("error creating temporary directory")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", errors.New("error reading file")
	}

	_, file := filepath.Split(filePath)
	destPath := filepath.Join(tempDir, file)

	if err := os.WriteFile(destPath, data, os.ModePerm); err != nil {
		return "", "", errors.New("error writing file to temporary directory")
	}

	volumeMap := fmt.Sprintf("%s:%s", tempDir, containerPath)
	return volumeMap, tempDir, nil
}

func runKicsScan(volumeMap, tempDir string) error {
	args := []string{
		"run", "--rm",
		"-v", volumeMap,
		"--name", viper.GetString(commonParams.KicsContainerNameKey),
		containerImage,
		"scan",
		"-p", containerPath,
		"-o", containerPath,
		"--report-formats", containerFormat,
	}

	logger.PrintIfVerbose("Starting kics container")
	logger.PrintIfVerbose("The report format and output path cannot be overridden.")

	out, err := exec.Command("docker", args...).CombinedOutput()
	logger.PrintIfVerbose(string(out))

	if err == nil {
		return nil
	}

	return handleKicsError(err, tempDir)
}

func handleKicsError(err error, tempDir string) error {
	msg := err.Error()
	code := extractErrorCode(msg)

	if slices.Contains(kicsErrorCodes, code) {
		results, readErr := readKicsResultsFile(tempDir)
		if readErr != nil {
			return errors.Errorf("%s", readErr)
		}
		if printErr := printKicsResults(&results); printErr != nil {
			return errors.Errorf("%s", printErr)
		}
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == util.EngineNoRunningCode {
		logger.PrintIfVerbose(msg)
		return errors.New(util.NotRunningEngineMessage)
	}

	if strings.Contains(msg, util.InvalidEngineError) || strings.Contains(msg, util.InvalidEngineErrorWindows) {
		logger.PrintIfVerbose(msg)
		return errors.New(util.InvalidEngineMessage)
	}

	return errors.Errorf("Check container engine state. Failed: %s", msg)
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

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}

func printKicsResults(results *wrappers.KicsResultsCollection) error {
	data, err := json.Marshal(results)
	if err != nil {
		return errors.Errorf("%s", err)
	}
	fmt.Println(string(data))
	return nil
}

func contains(filters []string, target string) bool {
	for _, f := range filters {
		if f != "" && strings.Contains(target, f) {
			return true
		}
	}
	return false
}
