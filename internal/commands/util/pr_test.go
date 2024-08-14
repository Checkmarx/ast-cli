package util

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	asserts "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
)

func TestNewPRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationGithub(nil, nil, nil)
	assert.Assert(t, cmd != nil, "PR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}

func TestNewMRDecorationCommandMustExist(t *testing.T) {
	cmd := PRDecorationGitlab(nil, nil, nil)
	assert.Assert(t, cmd != nil, "MR decoration command must exist")

	err := cmd.Execute()
	assert.ErrorContains(t, err, "scan-id")
}

func TestIsScanEnded(t *testing.T) {
	scansMockWrapper := &mock.ScansMockWrapper{Running: true}

	scanEnded, _ := IsScanEnded(scansMockWrapper, "ScanNotEnded")
	asserts.False(t, scanEnded)
	scansMockWrapper.Running = false
	scanEnded, _ = IsScanEnded(scansMockWrapper, "ScanEnded")
	asserts.True(t, scanEnded)
}
