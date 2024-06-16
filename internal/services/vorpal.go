package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/commands/vorpal/vorpalconfig"
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
}

func CreateVorpalScanRequest(vorpalParams VorpalScanParams) (*grpcs.ScanResult, error) {
	installedLatestVersion := false
	port, err := getVorpalPort()
	if err != nil {
		return nil, err
	}
	vorpalWrapper := grpcs.NewVorpalGrpcWrapper(port)

	if exists, _ := osinstaller.FileExists(vorpalParams.FilePath); !exists {
		return nil, fmt.Errorf("file %s does not exist", vorpalParams.FilePath)
	}

	vorpalInstalled, _ := osinstaller.FileExists(vorpalconfig.Params.ExecutableFilePath())
	if !vorpalInstalled {
		logger.PrintIfVerbose("Vorpal is not installed. Installing...")
		err = osinstaller.InstallOrUpgrade(&vorpalconfig.Params)
		installedLatestVersion = true
		if err != nil {
			return nil, err
		}
	}
	if vorpalParams.VorpalUpdateVersion && !installedLatestVersion {
		if !isLastVersion() {
			err = vorpalWrapper.HealthCheck()
			if err == nil {
				_ = vorpalWrapper.ShutDown()
			}
			err = osinstaller.InstallOrUpgrade(&vorpalconfig.Params)
			if err != nil {
				return nil, err
			}
		}
	}

	err = checkLicense(vorpalParams.IsDefaultAgent, vorpalParams.JwtWrapper)
	if err != nil {
		return nil, err
	}

	err = ensureVorpalServiceRunning(vorpalWrapper, port)
	if err != nil {
		return nil, err
	}

	sourceCode, err := readSourceCode(vorpalParams.FilePath)
	if err != nil {
		return nil, err
	}

	_, fileName := filepath.Split(vorpalParams.FilePath)
	return vorpalWrapper.Scan(fileName, sourceCode)
}

func getVorpalPort() (int, error) {
	port := viper.GetInt(params.VorpalPortKey)
	if port == 0 {
		var err error
		port, err = getAvailablePort()
		if err != nil {
			return 0, err
		}
		setConfigPropertyQuiet(params.VorpalPortKey, port)
	}
	return port, nil
}

func ensureVorpalServiceRunning(vorpalWrapper grpcs.VorpalWrapper, port int) error {
	if err := vorpalWrapper.HealthCheck(); err != nil {
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
		allowed, err := jwtWrapper.IsAllowedEngine(params.AIProtectionType)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("AI protection is not enabled for this user")
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

func getAvailablePort() (int, error) {
	port, err := getport.GetTcpPort()
	if err != nil {
		return 0, err
	}
	return port.Port, nil
}

func setConfigPropertyQuiet(propName string, propValue int) {
	viper.Set(propName, propValue)
	if viperErr := viper.SafeWriteConfig(); viperErr != nil {
		_ = viper.WriteConfig()
	}
}

func RunVorpalEngine(port int) error {
	dialTimeout := 2 * time.Second
	args := []string{
		"-listen",
		fmt.Sprintf("%d", port),
	}

	logger.PrintIfVerbose(fmt.Sprintf("Running vorpal engine with args: %v \n", args))

	cmd := exec.Command(vorpalconfig.Params.ExecutableFilePath(), args...)

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

func isLastVersion() bool {
	ans, err := osinstaller.IsLastVersion(vorpalconfig.Params.HashFilePath(), vorpalconfig.Params.HashDownloadURL, vorpalconfig.Params.HashFilePath())
	if err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("Error checking if the installation is up-to-date: %v", err))
		return false
	}
	return ans
}
