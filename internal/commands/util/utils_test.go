package util

import (
	"testing"

	"gotest.tools/assert"

	"github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
)

func TestNewUtilsCommand(t *testing.T) {
	logs := viper.GetString(params.LogsPathKey)
	logsWrapper := wrappers.NewLogsWrapper(logs)
	cmd := NewUtilsCommand(logsWrapper)
	assert.Assert(t, cmd != nil, "Utils command must exist")
}
