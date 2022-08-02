//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

const (
	utilsCommand                = "utils"
	remediationCommand          = "remediation"
	scaCommand                  = "sca"
	kicsCommand                 = "kics"
	packageFileFlag             = "package-file"
	packageFileValue            = "data/package.json"
	packageFileValueUnsupported = "data/package.jso"
	packageFlag                 = "package"
	packageValue                = "copyfiles"
	packageValueNotFound        = "copyfile"
	packageVersionFlag          = "package-version"
	packageVersionValue         = "200"
	resultsFileFlag             = "results-file"
	resultFileValue             = "data/results.json"
	kicsFileFlag                = "kics-files"
	kicsFileValue               = "data/"
	similarityIDFlag            = "similarity-ids"
	kicsEngine                  = "engine"
	similarityIDValue           = "9574288c118e8c87eea31b6f0b011295a39ec5e70d83fb70e839b8db4a99eba8"
	resultFileInvalidValue      = "./"
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

func TestKicsRemediation(t *testing.T) {
	_ = viper.BindEnv(pat)
	abs, _ := filepath.Abs(kicsFileValue)
	executeCmdNilAssertion(
		t,
		"Remediating kics result",
		utilsCommand,
		remediationCommand,
		kicsCommand,
		flag(kicsFileFlag),
		abs,
		flag(resultsFileFlag),
		resultFileValue,
	)
}

func TestKicsRemediationSimilarityFilter(t *testing.T) {
	_ = viper.BindEnv(pat)
	abs, _ := filepath.Abs(kicsFileValue)
	executeCmdNilAssertion(
		t,
		"Remediating kics result",
		utilsCommand,
		remediationCommand,
		kicsCommand,
		flag(kicsFileFlag),
		abs,
		flag(resultsFileFlag),
		resultFileValue,
		flag(similarityIDFlag),
		similarityIDValue,
	)
}

func TestKicsRemediationInvalidResults(t *testing.T) {
	abs, _ := filepath.Abs(kicsFileValue)
	args := []string{
		utilsCommand,
		remediationCommand,
		kicsCommand,
		flag(kicsFileFlag),
		abs,
		flag(resultsFileFlag),
		resultFileInvalidValue,
		flag(similarityIDFlag),
		similarityIDValue,
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "No results file was provided")
}

func TestKicsRemediationEngineFlag(t *testing.T) {
	_ = viper.BindEnv(pat)
	abs, _ := filepath.Abs(kicsFileValue)
	executeCmdNilAssertion(
		t,
		"Remediating kics result",
		utilsCommand,
		remediationCommand,
		kicsCommand,
		flag(kicsFileFlag),
		abs,
		flag(resultsFileFlag),
		resultFileValue,
		flag(kicsEngine),
		engineValue,
	)
}

func TestKicsRemediationInvalidEngine(t *testing.T) {
	abs, _ := filepath.Abs(kicsFileValue)
	args := []string{
		utilsCommand,
		remediationCommand,
		kicsCommand,
		flag(kicsFileFlag),
		abs,
		flag(resultsFileFlag),
		resultFileValue,
		flag(kicsEngine),
		invalidEngineValue,
	}

	err, _ := executeCommand(t, args...)
	assertError(t, err, "Please verify if engine is installed and running")
}
