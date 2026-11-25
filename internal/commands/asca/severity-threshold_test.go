package asca

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/spf13/cobra"
)

func newASCACmdWithThreshold(sourcePath, threshold string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String(commonParams.SourcesFlag, sourcePath, "")
	cmd.Flags().Bool(commonParams.ASCALatestVersion, false, "")
	cmd.Flags().String(commonParams.FormatFlag, printer.FormatJSON, "")
	cmd.Flags().String(commonParams.SeverityThresholdFlag, threshold, "")
	return cmd
}

func TestRunScanASCACommand_InvalidSeverityThreshold_FailsBeforeScan(t *testing.T) {
	for _, value := range []string{"hgh", "critical,high"} {
		err := RunScanASCACommand(&mock.JWTMockWrapper{})(newASCACmdWithThreshold("data/python-vul-file.py", value), []string{})
		if err == nil {
			t.Fatalf("expected an error for severity-threshold %q", value)
		}
		if !strings.Contains(err.Error(), "invalid severity-threshold value") {
			t.Fatalf("unexpected error message for %q: %v", value, err)
		}
		if !strings.Contains(err.Error(), "single level: critical, high, medium, low or info") {
			t.Fatalf("error should list the accepted levels: %v", err)
		}
	}
}

func TestRunScanASCACommand_EmptySeverityThreshold_TreatedAsNoThreshold(t *testing.T) {
	err := RunScanASCACommand(&mock.JWTMockWrapper{})(newASCACmdWithThreshold("", "   "), []string{})
	if err != nil {
		t.Fatalf("whitespace-only severity-threshold must not error, got: %v", err)
	}
}
