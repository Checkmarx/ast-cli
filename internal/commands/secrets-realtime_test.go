package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

func TestRunScanSecretsRealtimeCommand_TxtFile_ScanSuccess(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	execCmdNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--ignored-file-path", "",
	)
}

func TestRunScanSecretsRealtimeCommand_EmptyFilePath_ScanFailed(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	err := execCmdNotNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "", "--ignored-file-path", "",
	)
	assert.NotNil(t, err)
}

func TestRunScanSecretsRealtimeCommand_FFDisable_ScanFailed(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: false}
	err := execCmdNotNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--ignored-file-path", "",
	)
	assert.NotNil(t, err)
}
