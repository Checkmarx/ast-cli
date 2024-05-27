//go:build windows

package sastconfiguration

import (
	"github.com/checkmarx/ast-cli/internal/services/osinstaller"
)

var Params = osinstaller.InstallationConfiguration{
	ExecutableFile:  "cxcodeprobe.exe",
	DownloadURL:     "https://cxdownloadirelandteam17.s3.eu-west-1.amazonaws.com/cxcodeprobe-binary/cxcodeprobe_latest.zip",
	HashDownloadURL: "https://cxdownloadirelandteam17.s3.eu-west-1.amazonaws.com/cxcodeprobe-binary/hash.txt",
	FileName:        "cxcodeprobe.zip",
	HashFileName:    "hash.txt",
	WorkingDirName:  "SastRealtime",
}
