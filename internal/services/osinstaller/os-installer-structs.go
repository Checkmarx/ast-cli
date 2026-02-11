package osinstaller

import (
	"os"
	"path/filepath"
)

type InstallationConfiguration struct {
	ExecutableFile  string
	DownloadURL     string
	HashDownloadURL string
	FileName        string
	HashFileName    string
	WorkingDirName  string
}

func (i *InstallationConfiguration) ExecutableFilePath() string {
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
