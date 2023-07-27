package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

func TestMaskSecret(t *testing.T) {
	cmd := NewMaskSecretsCommand(mock.ChatMockWrapper{})
	cmd.SetArgs([]string{"utils", "mask", "--result-file", packageFileValue})
	err := cmd.Execute()
	assert.Assert(t, err == nil)
}
