//go:build linux && (arm64 || arm)

package ascaconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "ASCA_linux_arm64",
	DownloadURL:     "https://download.checkmarx.com/ASCA-binary/ASCA_linux_arm64.tar.gz",
	HashDownloadURL: "https://download.checkmarx.com/ASCA-binary/hash.txt",
	FileName:        "ASCA.tar.gz",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxASCA",
}
