//go:build integration

package integration

const (
	maskCommand           = "mask"
	resultsFileValue      = "data/package.json"
	resultFileNonExisting = "data/package.jso"
)

//func TestMaskSecrets(t *testing.T) {
//	executeCmdNilAssertion(
//		t,
//		"Remediating kics result",
//		utilsCommand,
//		maskCommand,
//		flag(params.ChatKicsResultFile),
//		resultsFileValue,
//	)
//}
//
//func TestFailedMaskSecrets(t *testing.T) {
//	args := []string{
//		utilsCommand,
//		maskCommand,
//		flag(params.ChatKicsResultFile),
//		resultFileNonExisting,
//	}
//	err, _ := executeCommand(t, args...)
//	assertError(t, err, "Error opening file : open data/package.jso: no such file or directory")
//}
