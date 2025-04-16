//go:build !integration

package commands

import (
	"testing"
)

func TestEnginesHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "engines")
}

func TestEnginesSub(t *testing.T) {
	execCmdNilAssertion(t, "engines")
}

func TestGetAllEngineAPIs(t *testing.T) {
	execCmdNilAssertion(t, "engines", "list-api")
}

func TestGetSASTEngineAPIs(t *testing.T) {
	execCmdNilAssertion(t, "engines", "list-api", "--engine-name", "sast")
}

func TestGetSCAEngineAPIs(t *testing.T) {
	execCmdNilAssertion(t, "engines", "list-api", "--engine-name", "sast")
}

func TestGetEngineAPIsWithNonExistFlag(t *testing.T) {
	execCmdNilAssertion(t, "engines", "list-api", "--engine-name", "abc")
}
