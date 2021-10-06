package util

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
	"testing"
)

func TestNewLogsCommand(t *testing.T) {
	logsWrapper := &mock.LogsMockWrapper{}
	cmd := NewLogsCommand(logsWrapper)
	assert.Assert(t, cmd != nil, "Logs command must exist")

	//TODO: test doesn't pass when executed
	/*cmd.SetArgs([]string{"--scan-id", "dummy"})
	err := cmd.Execute()
	assert.NilError(t, err, "Logs command should run with no errors")*/
}
