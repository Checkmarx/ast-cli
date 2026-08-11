package osinstaller

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type InstallationConfiguration struct {
	ExecutableFile  string
	DownloadURL     string
	HashDownloadURL string
	FileName        string
	HashFileName    string
	WorkingDirName  string
	// Vorpal: per-artifact checksum URL for binary verification
	ArchiveChecksumDownloadURL string
	ArchiveChecksumFileName    string
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

// BinaryFilePath returns the path to the downloaded archive on disk (before extraction).
func (i *InstallationConfiguration) BinaryFilePath() string {
	return filepath.Join(i.WorkingDir(), i.FileName)
}

// ArchiveChecksumFilePath is the local path for the optional per-artifact checksum file.
func (i *InstallationConfiguration) ArchiveChecksumFilePath() string {
	if i.ArchiveChecksumFileName == "" {
		return ""
	}
	return filepath.Join(i.WorkingDir(), i.ArchiveChecksumFileName)
}

// resolveArchiveChecksumVerification returns the local sha256sum path to verify against, and whether it must be downloaded first.
func (i *InstallationConfiguration) resolveArchiveChecksumVerification() (localPath string, needsExtraDownload bool, err error) {
	if i.ArchiveChecksumDownloadURL != "" {
		if i.ArchiveChecksumFileName == "" {
			return "", false, errors.New("ArchiveChecksumFileName is required when ArchiveChecksumDownloadURL is set")
		}
		return i.ArchiveChecksumFilePath(), true, nil
	}

	if strings.HasSuffix(i.HashFileName, ".sha256sum") {
		return i.HashFilePath(), false, nil
	}

	return "", false, nil
}
