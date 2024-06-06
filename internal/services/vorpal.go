package services

import (
	"os"
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
)

func CreateVorpalScanRequest(vorpalWrapper grpcs.VorpalWrapper, filePath string) (*grpcs.ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("Error reading file %v: %v", filePath, err)
		return nil, err
	}
	_, fileName := filepath.Split(filePath)
	sourceCode := string(data)
	return vorpalWrapper.Scan(fileName, sourceCode)
}
