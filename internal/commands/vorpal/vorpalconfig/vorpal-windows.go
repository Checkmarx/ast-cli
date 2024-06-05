//go:build windows

package vorpalconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "vorpal_windows_x64.exe",
	DownloadURL:     "https://download.checkmarx.com/vorpal-binary/vorpal_windows_x64.zip",
	HashDownloadURL: "https://download.checkmarx.com/vorpal-binary/hash.txt",
	FileName:        "vorpal.zip",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
