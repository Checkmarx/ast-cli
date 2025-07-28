package iacrealtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
)

// FileHandler handles file operations and environment preparation for IaC scanning
type FileHandler struct{}

// NewFileHandler creates a new FileHandler instance
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

func (fh *FileHandler) PrepareScanEnvironment(filePath string) (volumeMap, tempDir string, err error) {
	if !fh.HasSupportedExtension(filePath) {
		return "", "", errorconstants.NewRealtimeEngineError("Provided file is not supported by iac-realtime").Error()
	}

	tempDir, err = fh.CreateTempDirectory()
	if err != nil {
		return "", "", err
	}

	if err := fh.CopyFileToTempDir(filePath, tempDir); err != nil {
		return "", "", err
	}

	volumeMap = fmt.Sprintf("%s:%s", tempDir, ContainerPath)
	return volumeMap, tempDir, nil
}

func (fh *FileHandler) CreateTempDirectory() (string, error) {
	tempDir, err := os.MkdirTemp("", ContainerTempDirPattern)
	if err != nil {
		return "", errorconstants.NewRealtimeEngineError("error creating temporary directory").Error()
	}
	return tempDir, nil
}

func (fh *FileHandler) CopyFileToTempDir(filePath, tempDir string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errorconstants.NewRealtimeEngineError("error reading file").Error()
	}

	_, fileName := filepath.Split(filePath)

	fileName = filepath.Base(fileName)
	if fileName == "." || fileName == ".." || fileName == "" {
		return errorconstants.NewRealtimeEngineError("error writing file to temporary directory").Error()
	}

	destPath := filepath.Join(tempDir, fileName)

	if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(tempDir)) {
		return errorconstants.NewRealtimeEngineError("error writing file to temporary directory").Error()
	}

	if err := os.WriteFile(destPath, data, os.ModePerm); err != nil {
		return errorconstants.NewRealtimeEngineError("error writing file to temporary directory").Error()
	}

	return nil
}

func (fh *FileHandler) HasSupportedExtension(filePath string) bool {
	for _, filter := range commonParams.KicsBaseFilters {
		if filter != "" && strings.Contains(filePath, filter) {
			return true
		}
	}
	return false
}

func (fh *FileHandler) CleanupTempDirectory(tempDir string) error {
	if tempDir != "" {
		return os.RemoveAll(tempDir)
	}
	return nil
}
