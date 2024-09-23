//go:build darwin && amd64

package ascaconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "vorpal_darwin_x64",
	DownloadURL:     "https://download.checkmarx.com/vorpal-binary/vorpal_darwin_x64.tar.gz",
	HashDownloadURL: "https://download.checkmarx.com/vorpal-binary/hash.txt",
	FileName:        "vorpal.tar.gz",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxVorpal",
}
