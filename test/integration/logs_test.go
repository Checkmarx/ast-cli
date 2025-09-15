//go:build integration

package integration

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_DownloadScan_Logs_Success(t *testing.T) {
	args := []string{
		"scan", "create",
		flag(commonParams.ProjectName), GenerateRandomProjectNameForScan(),
		flag(commonParams.SourcesFlag), "data/sources.zip",
		flag(commonParams.ScanTypes), commonParams.SastType,
		flag(commonParams.BranchFlag), "dummy_branch",
		flag(commonParams.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, _ := executeCreateScan(t, args)
	args1 := []string{
		"scan", "logs", flag(commonParams.ScanIDFlag), scanID, flag(commonParams.ScanTypeFlag), commonParams.SastType,
	}
	err, _ := executeCommand(t, args1...)
	assert.Nil(t, err)

}

func Test_DownloadScan_Logs_Failed(t *testing.T) {
	args1 := []string{
		"scan", "logs", flag(commonParams.ScanIDFlag), "fake-scan-id", flag(commonParams.ScanTypeFlag), commonParams.SastType,
	}
	err, _ := executeCommand(t, args1...)
	assert.Error(t, err, "failed to download log")
}
