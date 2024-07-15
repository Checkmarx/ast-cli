package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfig"
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
	FilePathNotProvided = "File path not provided, Vorpal engine is running successfully."
	FileNotFound        = "File %s not found"
)

type VorpalScanParams struct {
	FilePath            string
	VorpalUpdateVersion bool
	IsDefaultAgent      bool
}

type VorpalWrappersParam struct {
	JwtWrapper          wrappers.JWTWrapper
	FeatureFlagsWrapper wrappers.FeatureFlagsWrapper
	VorpalWrapper       grpcs.VorpalWrapper
}

func CreateVorpalScanRequest(vorpalParams VorpalScanParams, wrapperParams VorpalWrappersParam) (*grpcs.ScanResult, error) {
	var err error
	if err != nil {
		return nil, err
	}

	err = manageVorpalInstallation(vorpalParams, wrapperParams.VorpalWrapper)
	if err != nil {
		return nil, err
	}

	err = ensureVorpalServiceRunning(wrapperParams, vorpalParams)
	if err != nil {
		return nil, err
	}

	emptyResults := validateFilePath(vorpalParams.FilePath)
	if emptyResults != nil {
		return emptyResults, nil
	}

	return executeScan(wrapperParams.VorpalWrapper, vorpalParams.FilePath)
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

func executeScan(vorpalWrapper grpcs.VorpalWrapper, filePath string) (*grpcs.ScanResult, error) {
	sourceCode, err := readSourceCode(filePath)
	if err != nil {
		return nil, err
	}

	_, fileName := filepath.Split(filePath)
	return vorpalWrapper.Scan(fileName, sourceCode)
}

func manageVorpalInstallation(vorpalParams VorpalScanParams, vorpalWrapper grpcs.VorpalWrapper) error {
	vorpalInstalled, _ := osinstaller.FileExists(vorpalconfig.Params.ExecutableFilePath())

	if vorpalParams.VorpalUpdateVersion || !vorpalInstalled {
		if err := vorpalWrapper.HealthCheck(); err == nil {
			_ = vorpalWrapper.ShutDown()
		}
		if err := osinstaller.InstallOrUpgrade(&vorpalconfig.Params); err != nil {
			return err
		}
	}
	return nil
}

func findVorpalPort() (int, error) {
	port, err := getAvailablePort()
	if err != nil {
		return 0, err
	}
	setConfigPropertyQuiet(params.VorpalPortKey, port)
	return port, nil
}

func getAvailablePort() (int, error) {
	port, err := getport.GetTcpPort()
	if err != nil {
		return 0, err
	}
	return port.Port, nil
}

func configureVorpalWrapper(existingVorpalWrapper grpcs.VorpalWrapper) (grpcs.VorpalWrapper, error) {
	if err := existingVorpalWrapper.HealthCheck(); err != nil {
		port, portErr := findVorpalPort()
		if portErr != nil {
			return nil, portErr
		}
		existingVorpalWrapper.ConfigurePort(port)
	}
	return existingVorpalWrapper, nil
}

func setConfigPropertyQuiet(propName string, propValue int) {
	viper.Set(propName, propValue)
	if viperErr := viper.SafeWriteConfig(); viperErr != nil {
		_ = viper.WriteConfig()
	}
}

func ensureVorpalServiceRunning(wrappersParam VorpalWrappersParam, vorpalParams VorpalScanParams) error {
	if err := wrappersParam.VorpalWrapper.HealthCheck(); err != nil {
		err = checkLicense(vorpalParams.IsDefaultAgent, wrappersParam)
		if err != nil {
			return err
		}
		wrappersParam.VorpalWrapper, err = configureVorpalWrapper(wrappersParam.VorpalWrapper)
		if err != nil {
			return err
		}
		if err := RunVorpalEngine(wrappersParam.VorpalWrapper.GetPort()); err != nil {
			return err
		}

		if err := wrappersParam.VorpalWrapper.HealthCheck(); err != nil {
			return err
		}
	}
	return nil
}

func checkLicense(isDefaultAgent bool, wrapperParams VorpalWrappersParam) error {
	if !isDefaultAgent {
		allowed, err := wrapperParams.JwtWrapper.IsAllowedEngine(params.AIProtectionType, wrapperParams.FeatureFlagsWrapper)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("%v", errorconstants.NoVorpalLicense)
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

func RunVorpalEngine(port int) error {
	dialTimeout := 5 * time.Second
	args := []string{
		"-listen",
		fmt.Sprintf("%d", port),
	}

	logger.PrintIfVerbose(fmt.Sprintf("Running vorpal engine with args: %v \n", args))

	cmd := exec.Command(vorpalconfig.Params.ExecutableFilePath(), args...)

	osinstaller.ConfigureIndependentProcess(cmd)

	err := cmd.Start()
	if err != nil {
		return err
	}

	ready := waitForServer(fmt.Sprintf("localhost:%d", port), dialTimeout)
	if !ready {
		return fmt.Errorf("server did not become ready in time")
	}

	logger.PrintIfVerbose("Vorpal engine started successfully!")

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
