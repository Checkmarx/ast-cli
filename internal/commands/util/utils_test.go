package util

import (
	"testing"

	"gotest.tools/assert"
)

const mockFormatErrorMessage = "Invalid format MOCK"

func TestNewUtilsCommand(t *testing.T) {
	cmd := NewUtilsCommand(nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Assert(t, cmd != nil, "Utils command must exist")
}
