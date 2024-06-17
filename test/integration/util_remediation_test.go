//go:build integration

package integration

const (
	utilsCommand                = "utils"
	remediationCommand          = "remediation"
	scaCommand                  = "sca"
	kicsCommand                 = "kics"
	packageFileValue            = "data/package.json"
	packageFileValueUnsupported = "data/package.jso"
	packageValue                = "copyfiles"
	packageValueNotFound        = "copyfile"
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

//func TestScaRemediation(t *testing.T) {
//	_ = viper.BindEnv(pat)
//	executeCmdNilAssertion(
//		t,
//		"Remediating sca result",
//		utilsCommand,
//		remediationCommand,
//		scaCommand,
//		flag(params.RemediationFiles),
//		packageFileValue,
//		flag(params.RemediationPackage),
//		packageValue,
//		flag(params.RemediationPackageVersion),
//		packageVersionValue,
//	)
//}

//func TestScaRemediationUnsupported(t *testing.T) {
//	args := []string{
//		utilsCommand,
//		remediationCommand,
//		scaCommand,
//		flag(params.RemediationFiles),
//		packageFileValueUnsupported,
//		flag(params.RemediationPackage),
//		packageValue,
//		flag(params.RemediationPackageVersion),
//		packageVersionValue,
//	}
//
//	err, _ := executeCommand(t, args...)
//	assertError(t, err, "Unsupported package manager file")
//}
//
//func TestScaRemediationNotFound(t *testing.T) {
//	args := []string{
//		utilsCommand,
//		remediationCommand,
//		scaCommand,
//		flag(params.RemediationFiles),
//		packageFileValue,
//		flag(params.RemediationPackage),
//		packageValueNotFound,
//		flag(params.RemediationPackageVersion),
//		packageVersionValue,
//	}
//
//	err, _ := executeCommand(t, args...)
//	assertError(t, err, "Package copyfile not found")
//}
//
//func TestKicsRemediation(t *testing.T) {
//	_ = viper.BindEnv(pat)
//	abs, _ := filepath.Abs(kicsFileValue)
//	executeCmdNilAssertion(
//		t,
//		"Remediating kics result",
//		utilsCommand,
//		remediationCommand,
//		kicsCommand,
//		flag(kicsFileFlag),
//		abs,
//		flag(resultsFileFlag),
//		resultFileValue,
//	)
//}
//
//func TestKicsRemediationSimilarityFilter(t *testing.T) {
//	_ = viper.BindEnv(pat)
//	abs, _ := filepath.Abs(kicsFileValue)
//	executeCmdNilAssertion(
//		t,
//		"Remediating kics result",
//		utilsCommand,
//		remediationCommand,
//		kicsCommand,
//		flag(kicsFileFlag),
//		abs,
//		flag(resultsFileFlag),
//		resultFileValue,
//		flag(similarityIDFlag),
//		similarityIDValue,
//	)
//}
//
//func TestKicsRemediationInvalidResults(t *testing.T) {
//	abs, _ := filepath.Abs(kicsFileValue)
//	args := []string{
//		utilsCommand,
//		remediationCommand,
//		kicsCommand,
//		flag(kicsFileFlag),
//		abs,
//		flag(resultsFileFlag),
//		resultFileInvalidValue,
//		flag(similarityIDFlag),
//		similarityIDValue,
//	}
//
//	err, _ := executeCommand(t, args...)
//	assertError(t, err, "No results file was provided")
//}
//
//func TestKicsRemediationEngineFlag(t *testing.T) {
//	_ = viper.BindEnv(pat)
//	abs, _ := filepath.Abs(kicsFileValue)
//	executeCmdNilAssertion(
//		t,
//		"Remediating kics result",
//		utilsCommand,
//		remediationCommand,
//		kicsCommand,
//		flag(kicsFileFlag),
//		abs,
//		flag(resultsFileFlag),
//		resultFileValue,
//		flag(kicsEngine),
//		engineValue,
//	)
//}
//
//func TestKicsRemediationInvalidEngine(t *testing.T) {
//	abs, _ := filepath.Abs(kicsFileValue)
//	args := []string{
//		utilsCommand,
//		remediationCommand,
//		kicsCommand,
//		flag(kicsFileFlag),
//		abs,
//		flag(resultsFileFlag),
//		resultFileValue,
//		flag(kicsEngine),
//		invalidEngineValue,
//	}
//	err, _ := executeCommand(t, args...)
//	assertError(t, err, util.InvalidEngineMessage)
//}
