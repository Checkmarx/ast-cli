package services

import (
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	getport "github.com/jsumners/go-getport"
)

func CreateVorpalScanRequest(vorpalWrapper grpcs.VorpalWrapper, filePath string) (*grpcs.ScanResult, error) {
	_, err := wrappers.GetAccessToken()
	//check vorpal engine env port set, if not set, set it
	//check vorpal engine zip installed and latest, if not install latest
	//check vorpal engine running, if not start it
	//check vorpal engine health, if not healthy, restart it
	healthCheckErr := vorpalWrapper.HealthCheck()
	if healthCheckErr != nil {
		_ = vorpalWrapper.ShutDown()
	}

	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %v: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return vorpalWrapper.Scan(fileName, sourceCode)
}

func getAvailablePort() (int, error) {
	port, err := getport.GetTcpPort()
	if err != nil {
		return 0, err
	}
	return port.Port, nil
}
