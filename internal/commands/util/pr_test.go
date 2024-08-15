package util

import (
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	asserts "github.com/stretchr/testify/assert"
	"testing"

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

func TestIfScanRunning_WhenScanRunning_ShouldReturnTrue(t *testing.T) {
	scansMockWrapper := &mock.ScansMockWrapper{Running: true}

	scanRunning, _ := isScanRunningOrQueued(scansMockWrapper, "ScanRunning")
	asserts.True(t, scanRunning)
}

func TestIfScanRunning_WhenScanRunning_ShouldReturnFalse(t *testing.T) {
	scansMockWrapper := &mock.ScansMockWrapper{Running: false}

	scanRunning, _ := isScanRunningOrQueued(scansMockWrapper, "ScanNotRunning")
	asserts.False(t, scanRunning)
}
