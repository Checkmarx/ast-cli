//go:build windows

package vorpalconfiguration

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "cxcodeprobe_windows_x64.exe",
	DownloadURL:     "https://download.checkmarx.com/cxcodeprobe-binary/cxcodeprobe_windows_x64.zip",
	HashDownloadURL: "https://download.checkmarx.com/cxcodeprobe-binary/hash.txt",
	FileName:        "vorpal.zip",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
