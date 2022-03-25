package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Utils command must exist")
}
