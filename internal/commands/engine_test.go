//go:build !integration

package commands

import (
	"testing"
)

const ()

func TestEngineHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "engines")
}

func TestEngineListHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "engine", "list-api")
}
