//go:build darwin && arm64

package vorpalconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "vorpal_darwin_arm64",
	DownloadURL:     "https://download.checkmarx.com/vorpal-binary/vorpal_darwin_arm64.tar.gz",
	HashDownloadURL: "https://download.checkmarx.com/vorpal-binary/hash.txt",
	FileName:        "vorpal.tar.gz",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
