//go:build linux

package vorpalconfiguration

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "cxcodeprobe_linux_x64",
	DownloadURL:     "https://download.checkmarx.com/cxcodeprobe-binary/cxcodeprobe_linux_x64.tar.gz",
	HashDownloadURL: "https://download.checkmarx.com/cxcodeprobe-binary/hash.txt",
	FileName:        "vorpal.tar.gz",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
