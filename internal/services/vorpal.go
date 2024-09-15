package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/ASCA/ASCAconfig"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	getport "github.com/jsumners/go-getport"
	"github.com/spf13/viper"
)

const (
	FilePathNotProvided = "File path not provided, ASCA engine is running successfully."
	FileNotFound        = "File %s not found"
)

type ASCAScanParams struct {
	FilePath          string
	ASCAUpdateVersion bool
	IsDefaultAgent    bool
}

type ASCAWrappersParam struct {
	JwtWrapper          wrappers.JWTWrapper
	FeatureFlagsWrapper wrappers.FeatureFlagsWrapper
	ASCAWrapper         grpcs.ASCAWrapper
}

func CreateASCAScanRequest(ASCAParams ASCAScanParams, wrapperParams ASCAWrappersParam) (*grpcs.ScanResult, error) {
	err := manageASCAInstallation(ASCAParams, wrapperParams)
	if err != nil {
		return nil, err
	}

	err = ensureASCAServiceRunning(wrapperParams, ASCAParams)
	if err != nil {
		return nil, err
	}

	emptyResults := validateFilePath(ASCAParams.FilePath)
	if emptyResults != nil {
		return emptyResults, nil
	}

	return executeScan(wrapperParams.ASCAWrapper, ASCAParams.FilePath)
}

func validateFilePath(filePath string) *grpcs.ScanResult {
	if filePath == "" {
		logger.PrintIfVerbose(FilePathNotProvided)
		return &grpcs.ScanResult{
			Message: FilePathNotProvided,
		}
	}

	if exists, _ := osinstaller.FileExists(filePath); !exists {
		fileNotFoundMsg := fmt.Sprintf(FileNotFound, filePath)
		logger.PrintIfVerbose(fileNotFoundMsg)
		return &grpcs.ScanResult{
			Error: &grpcs.Error{
				Description: fileNotFoundMsg,
			},
		}
	}

	return nil
}

func executeScan(ASCAWrapper grpcs.ASCAWrapper, filePath string) (*grpcs.ScanResult, error) {
	sourceCode, err := readSourceCode(filePath)
	if err != nil {
		return nil, err
	}

	_, fileName := filepath.Split(filePath)
	return ASCAWrapper.Scan(fileName, sourceCode)
}

func manageASCAInstallation(ASCAParams ASCAScanParams, ASCAWrappers ASCAWrappersParam) error {
	ASCAInstalled, _ := osinstaller.FileExists(ASCAconfig.Params.ExecutableFilePath())

	if !ASCAInstalled || ASCAParams.ASCAUpdateVersion {
		if err := checkLicense(ASCAParams.IsDefaultAgent, ASCAWrappers); err != nil {
			_ = ASCAWrappers.ASCAWrapper.ShutDown()
			return err
		}
		newInstallation, err := osinstaller.InstallOrUpgrade(&ASCAconfig.Params)
		if err != nil {
			return err
		}
		if newInstallation {
			_ = ASCAWrappers.ASCAWrapper.ShutDown()
		}
	}
	return nil
}

func findASCAPort() (int, error) {
	port, err := getAvailablePort()
	if err != nil {
		return 0, err
	}
	setConfigPropertyQuiet(params.ASCAPortKey, port)
	return port, nil
}

func getAvailablePort() (int, error) {
	port, err := getport.GetTcpPort()
	if err != nil {
		return 0, err
	}
	return port.Port, nil
}

func configureASCAWrapper(existingASCAWrapper grpcs.ASCAWrapper) (grpcs.ASCAWrapper, error) {
	if err := existingASCAWrapper.HealthCheck(); err != nil {
		port, portErr := findASCAPort()
		if portErr != nil {
			return nil, portErr
		}
		existingASCAWrapper.ConfigurePort(port)
	}
	return existingASCAWrapper, nil
}

func setConfigPropertyQuiet(propName string, propValue int) {
	viper.Set(propName, propValue)
	if viperErr := viper.SafeWriteConfig(); viperErr != nil {
		_ = viper.WriteConfig()
	}
}

func ensureASCAServiceRunning(wrappersParam ASCAWrappersParam, ASCAParams ASCAScanParams) error {
	if err := wrappersParam.ASCAWrapper.HealthCheck(); err != nil {
		err = checkLicense(ASCAParams.IsDefaultAgent, wrappersParam)
		if err != nil {
			return err
		}
		wrappersParam.ASCAWrapper, err = configureASCAWrapper(wrappersParam.ASCAWrapper)
		if err != nil {
			return err
		}
		if err := RunASCAEngine(wrappersParam.ASCAWrapper.GetPort()); err != nil {
			return err
		}

		if err := wrappersParam.ASCAWrapper.HealthCheck(); err != nil {
			return err
		}
	}
	return nil
}

func checkLicense(isDefaultAgent bool, wrapperParams ASCAWrappersParam) error {
	if !isDefaultAgent {
		allowed, err := wrapperParams.JwtWrapper.IsAllowedEngine(params.AIProtectionType, wrapperParams.FeatureFlagsWrapper)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("%v", errorconstants.NoASCALicense)
		}
	}
	return nil
}

func readSourceCode(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %v: %v", filePath, err)
		return "", err
	}
	return string(data), nil
}

func RunASCAEngine(port int) error {
	dialTimeout := 5 * time.Second
	args := []string{
		"-listen",
		fmt.Sprintf("%d", port),
	}

	logger.PrintIfVerbose(fmt.Sprintf("Running ASCA engine with args: %v \n", args))

	cmd := exec.Command(ASCAconfig.Params.ExecutableFilePath(), args...)

	osinstaller.ConfigureIndependentProcess(cmd)

	err := cmd.Start()
	if err != nil {
		return err
	}

	ready := waitForServer(fmt.Sprintf("localhost:%d", port), dialTimeout)
	if !ready {
		return fmt.Errorf("server did not become ready in time")
	}

	logger.PrintIfVerbose("ASCA engine started successfully!")

	return nil
}

func waitForServer(address string, timeout time.Duration) bool {
	waitingDuration := 50 * time.Millisecond
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", address)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(waitingDuration)
	}
	return false
}
