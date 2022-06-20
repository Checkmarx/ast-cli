package util

import (
	"gotest.tools/assert"
	"testing"
)

func TestNewPRDecorationCommandMustExist(t *testing.T) {
	cmd := NewPRDecorationCommand(nil)
	assert.Assert(t, cmd != nil, "PR decoration command must exist")

	err := cmd.Execute()
	assert.Error(t, err, "failed creating PR Decoration: Please provide scan-id flag")
}
