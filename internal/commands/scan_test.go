package commands

import (
	"testing"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
)

func TestRunCreateCommandWithFile(t *testing.T) {
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	uploadsMockWrapper := &wrappers.UploadsMockWrapper{}
	runCommand := runCreateScanCommand("", "./payloads/uploads.json",
		"./sources/sources.zip", scansMockWrapper, uploadsMockWrapper)
	runCommand(nil, nil)
}
func TestRunCreateCommandWithInput(t *testing.T) {
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	uploadsMockWrapper := &wrappers.UploadsMockWrapper{}
	runCommand := runCreateScanCommand("{\"project\":{\"id\":\"test\",\"type\":\"upload\",\"handler\":"+
		"{\"url\":\"MOSHIKO\"},\"tags\":{}},\"config\":"+
		"[{\"type\":\"sast\",\"value\":{\"presetName\":\"Default\"}}],\"tags\":{}}",
		"",
		"./sources/sources.zip", scansMockWrapper, uploadsMockWrapper)
	runCommand(nil, nil)
}
func TestRunCreateCommandWithNoInput(t *testing.T) {
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	uploadsMockWrapper := &wrappers.UploadsMockWrapper{}
	runCommand := runCreateScanCommand("", "", "./sources/sources.zip", scansMockWrapper, uploadsMockWrapper)
	runCommand(nil, nil)
}

func TestRunGetScanByIdCommand(t *testing.T) {
	scansMockWrapper := &wrappers.ScansMockWrapper{}
	runCommand := runGetScanByIDCommand("MOCK", scansMockWrapper)
	runCommand(nil, nil)
}
