package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunScanOssRealtimeCommand_RequirementsTxtFile_ScanSuccess(t *testing.T) {
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/requirements.txt",
	)
}

func TestRunScanOssRealtimeCommand_EmptyFilePath_ScanFailed(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "",
	)
	assert.NotNil(t, err)
}

func TestRunScanOssRealtimeCommand_PackageJsonFile_ScanSuccess(t *testing.T) {
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json",
	)
}
func TestRunScanOssRealtimeCommand_UnsupportedFileType_ScanFailed(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "not-supported-extension.txt",
	)
	assert.NotNil(t, err)
}
