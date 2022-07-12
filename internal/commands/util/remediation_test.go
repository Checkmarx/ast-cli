package util

import (
	"testing"

	"gotest.tools/assert"
)

const (
	packageFileFlag             = "--package-file"
	packageFileValue            = "../data/package.json"
	packageFileValueUnsupported = "../data/package.jso"
	packageFlag                 = "--package"
	packageValue                = "copyfiles"
	packageValueNotFound        = "copyfile"
	packageVersionFlag          = "--package-version"
	packageVersionValue         = "200"
)

func TestNewRemediationCommandCommand(t *testing.T) {
	cmd := NewRemediationCommand()
	assert.Assert(t, cmd != nil, "Remediation command must exist")
}

func TestRemediationScaCommandCommand(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(cmd, packageFileFlag, packageFileValue, packageFlag, packageValue, packageVersionFlag, packageVersionValue)
	assert.Assert(t, err == nil, "Remediation command must pass")
}

func TestRemediationScaCommandCommandUnsupported(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(cmd, packageFileFlag, packageFileValueUnsupported, packageFlag, packageValue, packageVersionFlag, packageVersionValue)
	assert.Assert(t, err != nil, "Unsuported package manager file")
}

func TestRemediationScaCommandCommandPackageNotFound(t *testing.T) {
	cmd := RemediationScaCommand()
	err := executeTestCommand(cmd, packageFileFlag, packageFileValue, packageFlag, packageValueNotFound, packageVersionFlag, packageVersionValue)
	assert.Assert(t, err != nil, "Package not found")
}
