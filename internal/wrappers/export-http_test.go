package wrappers

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
)

func TestExportHTTPWrapper_GetExportPackage(t *testing.T) {
	configuration.LoadConfiguration()
	wrapper := ExportHTTPWrapper{ExportPath: "api/sca/export/requests"}
	exportID, err := wrapper.GetExportPackage("85c134bb-3ac4-40a4-bac7-f3fe27750232")
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if exportID != nil {
		fmt.Println("ExportID: ", exportID)
	}
}
