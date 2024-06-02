//go:build darwin

package vorpalconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "cxcodeprobe_darwin_x64",
	DownloadURL:     "https://download.checkmarx.com/cxcodeprobe-binary/cxcodeprobe_darwin_x64.tar.gz",
	HashDownloadURL: "https://download.checkmarx.com/cxcodeprobe-binary/hash.txt",
	FileName:        "vorpal.tar.gz",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
