package util

import (
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

const (
	packageFileFlag             = "--package-files"
	packageFileValue            = "../data/package.json"
	packageFileValueUnsupported = "../data/package.jso"
	packageFlag                 = "--package"
	packageValue                = "copyfiles"
	packageValueNotFound        = "copyfile"
	packageVersionFlag          = "--package-version"
	packageVersionValue         = "200"
	resultsFileFlag             = "--results-file"
	resultFileValue             = "../data/results.json"
	invalidResultFileValue      = "../"
	kicsFileFlag                = "--kics-files"
	kicsFileValue               = "../data/"
	engineFlag                  = "--engine"
	engineValue                 = "docker"
	similarityIDFlag            = "--similarity-ids"
	similarityIDValue           = "b42a19486a8e18324a9b2c06147b1c49feb3ba39a0e4aeafec5665e60f98d047,9574288c118e8c87eea31b6f0b011295a39ec5e70d83fb70e839b8db4a99eba8"
	invalidEngineValue          = "invalidEngine"
)

func TestNewRemediationCommand(t *testing.T) {
	cmd := NewRemediationCommand()
	assert.Assert(t, cmd != nil, "Remediation command must exist")
}

func TestRemediationScaCommand(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(
		cmd,
		packageFileFlag,
		packageFileValue,
		packageFlag,
		packageValue,
		packageVersionFlag,
		packageVersionValue,
	)
	assert.Assert(t, err == nil, "Remediation command must pass")
}

func TestRemediationScaCommandUnsupported(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(
		cmd,
		packageFileFlag,
		packageFileValueUnsupported,
		packageFlag,
		packageValue,
		packageVersionFlag,
		packageVersionValue,
	)
	assert.Assert(t, err != nil, "Unsuported package manager file")
}

func TestRemediationScaCommandPackageNotFound(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(
		cmd,
		packageFileFlag,
		packageFileValue,
		packageFlag,
		packageValueNotFound,
		packageVersionFlag,
		packageVersionValue,
	)
	assert.Assert(t, err != nil, "Package not found")
}

func TestRemediationKicsCommand(t *testing.T) {
	cmd := RemediationKicsCommand()
	abs, _ := filepath.Abs(kicsFileValue)
	err := executeTestCommand(cmd, resultsFileFlag, resultFileValue, kicsFileFlag, abs)
	assert.Assert(t, err == nil, "Remediation command must pass")
}

func TestRemediationKicsCommandInvalidResults(t *testing.T) {
	cmd := RemediationKicsCommand()
	abs, _ := filepath.Abs(kicsFileValue)
	err := executeTestCommand(cmd, resultsFileFlag, invalidResultFileValue, kicsFileFlag, abs)
	assert.Assert(t, err != nil, "No results file was provided")
}

func TestRemediationKicsCommandEngineFlag(t *testing.T) {
	cmd := RemediationKicsCommand()
	abs, _ := filepath.Abs(kicsFileValue)
	err := executeTestCommand(cmd, resultsFileFlag, resultFileValue, kicsFileFlag, abs, engineFlag, engineValue)
	assert.Assert(t, err == nil, "Remediation command must pass")
}

func TestRemediationKicsCommandInvalidEngine(t *testing.T) {
	cmd := RemediationKicsCommand()
	abs, _ := filepath.Abs(kicsFileValue)
	err := executeTestCommand(cmd, resultsFileFlag, resultFileValue, kicsFileFlag, abs, engineFlag, invalidEngineValue)
	assert.Assert(t, err != nil, InvalidEngineMessage)
}

func TestRemediationKicsCommandSimilarityFilter(t *testing.T) {
	cmd := RemediationKicsCommand()
	abs, _ := filepath.Abs(kicsFileValue)
	err := executeTestCommand(
		cmd,
		resultsFileFlag,
		resultFileValue,
		kicsFileFlag,
		abs,
		similarityIDFlag,
		similarityIDValue,
	)
	assert.Assert(t, err == nil, "Remediation command must pass")
}
