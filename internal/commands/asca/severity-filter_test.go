package asca

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
)

func TestRunScanASCACommand_InvalidSeverityFilter_FailsBeforeScan(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "data/python-vul-file.py", "")
	cmd.Flags().Bool(commonParams.ASCALatestVersion, false, "")
	cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
	cmd.Flags().String(commonParams.SeverityFilterFlag, "critical,hgh", "")

	err := RunScanASCACommand(&mock.JWTMockWrapper{})(cmd, []string{})
	if err == nil {
		t.Fatal("expected an error for an invalid severity-filter value")
	}
	if !strings.Contains(err.Error(), "invalid severity-filter value(s): hgh") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "critical, high, medium, low, info") {
		t.Fatalf("error should list accepted values: %v", err)
	}
}

func TestRunScanASCACommand_EmptySeverityFilter_TreatedAsNoFilter(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, "", "")
	cmd.Flags().Bool(commonParams.ASCALatestVersion, false, "")
	cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
	cmd.Flags().String(commonParams.SeverityFilterFlag, "   ", "")

	err := RunScanASCACommand(&mock.JWTMockWrapper{})(cmd, []string{})
	if err != nil {
		t.Fatalf("whitespace-only severity-filter must not error, got: %v", err)
	}
}
