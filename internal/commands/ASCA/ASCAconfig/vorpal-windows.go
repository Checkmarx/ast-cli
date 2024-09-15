//go:build windows

package ASCAconfig

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "ASCA_windows_x64.exe",
	DownloadURL:     "https://download.checkmarx.com/ASCA-binary/ASCA_windows_x64.zip",
	HashDownloadURL: "https://download.checkmarx.com/ASCA-binary/hash.txt",
	FileName:        "ASCA.zip",
	HashFileName:    "hash.txt",
	WorkingDirName:  "CxASCA",
}
