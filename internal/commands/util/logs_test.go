package util

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/checkmarxDev/ast-cli/internal/wrappers/mock"
)

func TestNewLogsCommand(t *testing.T) {
	logsWrapper := &mock.LogsMockWrapper{}
	cmd := NewLogsCommand(logsWrapper)
	assert.Assert(t, cmd != nil, "Logs command must exist")

	err := executeTestCommand(cmd, "--scan-id", "dummy", "--scan-type", "sast")
	assert.NilError(t, err)
	fmt.Println(err)
}
