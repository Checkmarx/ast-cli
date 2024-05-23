package osinstaller

import (
	"os"
	"path/filepath"
)

type InstallableRealTime struct {
	ExecutableFile  string
	DownloadURL     string
	HashDownloadURL string
	FileName        string
	HashFileName    string
	WorkingDirName  string
}

func (i *InstallableRealTime) ExecutableFilePath() string {
	return filepath.Join(os.TempDir(), i.WorkingDirName, i.ExecutableFile)
}

func (i *InstallableRealTime) HashFilePath() string {
	return filepath.Join(os.TempDir(), i.WorkingDirName, i.HashFileName)
}

func (i *InstallableRealTime) WorkingDir() string {
	return filepath.Join(os.TempDir(), i.WorkingDirName)
}
