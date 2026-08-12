//go:build !integration

package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMCPCommand_Metadata(t *testing.T) {
	cmd := NewMCPCommand("1.0.0", func() bool { return true })
	assert.Equal(t, "mcp", cmd.Use)

	bridgeCmd, _, err := cmd.Find([]string{"bridge"})
	assert.NoError(t, err)
	assert.Equal(t, "bridge", bridgeCmd.Use)
}
