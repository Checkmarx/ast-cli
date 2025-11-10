//go:build integration

package integration

import (
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSecrets_RealtimeScan_TextFile_Success(t *testing.T) {
	args := []string{
		"scan", "secrets-realtime", "-s", "data/secret-exposed.txt", flag(commonParams.IgnoredFilePathFlag), "",
	}
	err, _ := executeCommand(t, args...)
	assert.Nil(t, err)
}

func TestSecrets_RealtimeScan_Empty_filePath_Fail(t *testing.T) {
	args := []string{
		"scan", "secrets-realtime", "-s", "", flag(commonParams.IgnoredFilePathFlag), "",
	}
	err, _ := executeCommand(t, args...)
	assert.NotNil(t, err)
}
