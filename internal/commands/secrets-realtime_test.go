package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestRunScanSecretsRealtimeCommand_TxtFile_ScanSuccess(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt",
	)
}

func TestRunScanSecretsRealtimeCommand_EmptyFilePath_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "",
	)
	assert.NotNil(t, err)
}

func TestRunScanSecretsRealtimeCommand_FFDisable_ScanFailed(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: false}
	err := execCmdNotNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt",
	)
	assert.NotNil(t, err)
}
