package osinstaller

import (
	"github.com/checkmarx/ast-cli/internal/logger"
	"os"
	"path/filepath"
)

type InstallationConfiguration struct {
	ExecutableFile   string
	DownloadURL      string
	HashDownloadURL  string
	FileName         string
	HashFileName     string
	WorkingDirName   string
	VorpalCustomPath string
}

func (i *InstallationConfiguration) SetVorpalCustomPath(path string) {
	i.VorpalCustomPath = path
}

func (i *InstallationConfiguration) ExecutableFilePath() string {
	if i.VorpalCustomPath != "" && i.WorkingDirName == "CxVorpal" {
		logger.PrintfIfVerbose("Using custom ASCA path: %s", i.VorpalCustomPath)
		return filepath.Join(i.VorpalCustomPath, i.ExecutableFile)
	}

	basePath := os.TempDir()
	homeDir, err := os.UserHomeDir()
	if err == nil {
		basePath = homeDir + "/.checkmarx/"
	}
	return filepath.Join(basePath, i.WorkingDirName, i.ExecutableFile)
}

func (i *InstallationConfiguration) HashFilePath() string {
	basePath := os.TempDir()
	homeDir, err := os.UserHomeDir()
	if err == nil {
		basePath = homeDir + "/.checkmarx/"
	}
	return filepath.Join(basePath, i.WorkingDirName, i.HashFileName)
}

func (i *InstallationConfiguration) WorkingDir() string {
	basePath := os.TempDir()
	homeDir, err := os.UserHomeDir()
	if err == nil {
		basePath = homeDir + "/.checkmarx/"
	}
	return filepath.Join(basePath, i.WorkingDirName)
}
