//go:build darwin

package scarealtime

import (
	"path/filepath"
)

var Params = ScaRealTime{
	ExecutableFilePath:         filepath.Join(ScaResolverWorkingDir, "ScaResolver"),
	HashFilePath:               filepath.Join(ScaResolverWorkingDir, "ScaResolver.tar.gz.sha256sum"),
	SCAResolverDownloadURL:     "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-macos64.tar.gz",
	SCAResolverHashDownloadURL: "https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-macos64.tar.gz.sha256sum",
	SCAResolverFileName:        "ScaResolver.tar.gz",
	SCAResolverHashFileName:    "ScaResolver.tar.gz.sha256sum",
}
