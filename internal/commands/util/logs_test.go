package util

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
	"testing"
)

func TestNewLogsCommand(t *testing.T) {
	logsWrapper := &mock.LogsMockWrapper{}
	cmd := NewLogsCommand(logsWrapper)
	assert.Assert(t, cmd != nil, "Logs command must exist")

	cmd.SetArgs([]string{"--scan-id", "dummy", "--scan-type", "sast"})
	err := cmd.Execute()
	fmt.Println(err)
}
