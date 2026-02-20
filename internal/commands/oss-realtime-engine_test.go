package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestRunScanOssRealtimeCommand_RequirementsTxtFile_ScanSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/requirements.txt",
	)
}

func TestRunScanOssRealtimeCommand_EmptyFilePath_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "",
	)
	assert.NotNil(t, err)
}

func TestRunScanOssRealtimeCommand_PackageJsonFile_ScanSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json",
	)
}

func TestRunScanOssRealtimeCommand_UnsupportedFileType_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "not-supported-extension.txt",
	)
	assert.NotNil(t, err)
}

func TestRunScanOssRealtimeCommand_SeverityThreshold_ScanSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", "Malicious,Critical",
	)
}

func TestRunScanOssRealtimeCommand_SeverityThresholdOtherValues_CaseInsensitive_ScanSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", "MAliCious,CRItICAL,HigH,Medium",
	)
}

func TestRunScanOssRealtimeCommand_SeverityThresholdFlagWithoutValue_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", "",
	)
	assert.NotNil(t, err)
}

func TestRunScanOssRealtimeCommand_SeverityThresholdWrongThresholdValues_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "oss-realtime", "-s", "data/manifests/package.json", "--severity-threshold", "a,b",
	)
	assert.NotNil(t, err)
}
