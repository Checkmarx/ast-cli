//go:build integration

package integration

//func TestRunGetBflByScanIdAndQueryId(t *testing.T) {
//
//	expectedMsg := "required flag(s) " + "\"" + params.QueryIDFlag + "\"" + ", " + "\"" + params.ScanIDFlag + "\"" + " not set"
//	assertRequiredParameter(t, expectedMsg, "results", "bfl")
//	scanID, _ := getRootScan(t)
//	queryID := "17765437696070740537"
//
//	outputBuffer := executeCmdNilAssertion(
//		t, "Getting BFL should pass.", "results", "bfl",
//		flag(params.ScanIDFlag), scanID,
//		flag(params.QueryIDFlag), queryID,
//		flag(params.FormatFlag), "json")
//
//	bflResult := []wrappers.ScanResultNode{}
//	_ = unmarshall(t, outputBuffer, &bflResult, "Reading BFL results should pass")
//
//}
//
//func TestRunGetBflWithInvalidScanIDandQueryID(t *testing.T) {
//
//	err, _ := executeCommand(
//		t, "results", "bfl",
//		flag(params.ScanIDFlag), "123456",
//		flag(params.QueryIDFlag), "abcd",
//		flag(params.FormatFlag), "json")
//
//	assertError(t, err, "Failed getting BFL: CODE: 5002, Failed getting BFL")
//}
