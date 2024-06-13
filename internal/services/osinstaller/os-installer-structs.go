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
	return filepath.Join(os.TempDir(), i.WorkingDirName, i.ExecutableFile)
}

func (i *InstallationConfiguration) HashFilePath() string {
	return filepath.Join(os.TempDir(), i.WorkingDirName, i.HashFileName)
}

func (i *InstallationConfiguration) WorkingDir() string {
	return filepath.Join(os.TempDir(), i.WorkingDirName)
}
