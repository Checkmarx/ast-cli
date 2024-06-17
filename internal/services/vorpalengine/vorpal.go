package vorpalengine

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

type VorpalScanParams struct {
	FilePath            string
	VorpalUpdateVersion bool
	IsDefaultAgent      bool
	JwtWrapper          wrappers.JWTWrapper
	FeatureFlagsWrapper wrappers.FeatureFlagsWrapper
	VorpalWrapper       grpcs.VorpalWrapper
}

func CreateVorpalScanRequest(vorpalParams VorpalScanParams) (*grpcs.ScanResult, error) {
	var err error
	vorpalParams.VorpalWrapper, err = configureVorpalWrapper(vorpalParams.VorpalWrapper)
	vorpalWrapper := vorpalParams.VorpalWrapper
	if err != nil {
		return nil, err
	}

	err = manageVorpalInstallation(vorpalParams)
	if err != nil {
		return nil, err
	}

	err = ensureVorpalServiceRunning(vorpalWrapper, vorpalWrapper.GetPort(), vorpalParams)
	if err != nil {
		return nil, err
	}

	if exists, _ := osinstaller.FileExists(vorpalParams.FilePath); !exists {
		logger.PrintIfVerbose(fmt.Sprintf("file %v not exist or not provided", vorpalParams.FilePath))
		return &grpcs.ScanResult{}, nil
	}

	sourceCode, err := readSourceCode(vorpalParams.FilePath)
	if err != nil {
		return nil, err
	}

	_, fileName := filepath.Split(vorpalParams.FilePath)
	return vorpalWrapper.Scan(fileName, sourceCode)
}

func manageVorpalInstallation(vorpalParams VorpalScanParams) error {
	vorpalInstalled, _ := osinstaller.FileExists(vorpalconfig.Params.ExecutableFilePath())

	if vorpalParams.VorpalUpdateVersion && !vorpalInstalled {
		if err := vorpalParams.VorpalWrapper.HealthCheck(); err == nil {
			_ = vorpalParams.VorpalWrapper.ShutDown()
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

func ensureVorpalServiceRunning(vorpalWrapper grpcs.VorpalWrapper, port int, vorpalParams VorpalScanParams) error {
	if err := vorpalWrapper.HealthCheck(); err != nil {
		err = checkLicense(vorpalParams.IsDefaultAgent, vorpalParams.JwtWrapper)
		if err != nil {
			return err
		}

		if err := RunVorpalEngine(port); err != nil {
			return err
		}

		if err := vorpalWrapper.HealthCheck(); err != nil {
			return err
		}
	}
	return nil
}

func checkLicense(isDefaultAgent bool, jwtWrapper wrappers.JWTWrapper) error {
	if !isDefaultAgent {
		// TODO: check enforcement
		allowed, err := jwtWrapper.IsAllowedEngine(params.AIProtectionType)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf(errorconstants.NoVorpalLicense)
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
	dialTimeout := 1 * time.Second
	args := []string{
		"-listen",
		fmt.Sprintf("%d", port),
	}

	logger.PrintIfVerbose(fmt.Sprintf("Running vorpal engine with args: %v \n", args))

	cmd := exec.Command(vorpalconfig.Params.ExecutableFilePath(), args...)

	configureIndependentProcess(cmd)

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
