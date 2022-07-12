package util

import (
	"testing"
)

//TODO: can we assert something?
func TestRootHelpFunc(t *testing.T) {
	cmd := NewConfigCommand()
	cmd.Long = ""
	cmd.SetArgs([]string{"show "})
	RootHelpFunc(cmd)
}
