//go:build !integration

package commands

import (
	"gotest.tools/assert"
	"testing"
)

const (
	unknownEngineFlag = "unknown flag: --chibutero"
)

func TestEngineNoSub(t *testing.T) {
	execCmdNilAssertion(t, "engine")
}

func TestGetAllEnginesAPI(t *testing.T) {
	execCmdNilAssertion(t, "engine", "list-api")
}

func TestRunGetAllEngineAPIFlagNonExist(t *testing.T) {
	err := execCmdNotNilAssertion(t, "engine", "list-api", "--chibutero")
	assert.Assert(t, err.Error() == unknownEngineFlag)
}

func TestGetSASTEnginesAPI(t *testing.T) {
	execCmdNilAssertion(t, "engine", "list-api", "--engine-name", "sast")
}

func TestGetSCAEngineAPI(t *testing.T) {
	execCmdNilAssertion(t, "engine", "list-api", "--engine-name", "sca")
}

func TestGetDASTEngineAPI(t *testing.T) {
	execCmdNilAssertion(t, "engine", "list-api", "--engine-name", "dast")
}
