package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunScanSecretsRealtimeCommand_TxtFile_ScanSuccess(t *testing.T) {
	clearFlags()
	execCmdNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt", "--ignored-file-path", "",
	)
}

func TestRunScanSecretsRealtimeCommand_EmptyFilePath_ScanFailed(t *testing.T) {
	clearFlags()
	err := execCmdNotNilAssertion(
		t,
		"scan", "secrets-realtime", "-s", "", "--ignored-file-path", "",
	)
	assert.NotNil(t, err)
}
