package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewPRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationGithub(nil)
	assert.Assert(t, cmd != nil, "PR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}
