//go:build integration

package integration

import (
	"testing"

	"github.com/spf13/viper"
)

const (
	utilsCommand = "utils"
	remediationCommand  = "remediation"
	scaCommand  = "sca"
	packageFileFlag             = "package-file"
	packageFileValue            = "data/package.json"
	packageFileValueUnsupported = "data/package.jso"
	packageFlag                 = "package"
	packageValue                = "copyfiles"
	packageValueNotFound        = "copyfile"
	packageVersionFlag          = "package-version"
	packageVersionValue         = "200"
)

func TestScaRemediation(t *testing.T) {
	_ = viper.BindEnv(pat)
	executeCmdNilAssertion(
		t,
		"Remediating sca result",
		utilsCommand,
		remediationCommand,
		scaCommand,
		flag(packageFileFlag),
		packageFileValue,
		flag(packageFlag),
		packageValue,
		flag(packageVersionFlag),
		packageVersionValue,
	)
}

func TestScaRemediationUnsupported(t *testing.T) {
	args := []string{
		utilsCommand,
		remediationCommand,
		scaCommand,
		flag(packageFileFlag),
		packageFileValueUnsupported,
		flag(packageFlag),
		packageValue,
		flag(packageVersionFlag),
		packageVersionValue,
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Unsupported package manager file")
}

func TestScaRemediationNotFound(t *testing.T) {
	args := []string{
		utilsCommand,
		remediationCommand,
		scaCommand,
		flag(packageFileFlag),
		packageFileValue,
		flag(packageFlag),
		packageValueNotFound,
		flag(packageVersionFlag),
		packageVersionValue,
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Package copyfile not found")
}