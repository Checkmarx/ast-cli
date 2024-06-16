package commands

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/cobra"
)

func runScanVorpalCommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		vorpalLatestVersion, _ := cmd.Flags().GetBool(commonParams.VorpalLatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		agent, _ := cmd.PersistentFlags().GetString(commonParams.AgentFlag)
		vorpalParams := services.VorpalScanParams{
			FilePath:            fileSourceFlag,
			VorpalUpdateVersion: vorpalLatestVersion,
			IsDefaultAgent:      agent == commonParams.DefaultAgent,
			JwtWrapper:          jwtWrapper,
			FeatureFlagsWrapper: featureFlagsWrapper,
		}
		scanResult, err := services.CreateVorpalScanRequest(vorpalParams)
		if err != nil {
			return err
		}

		err = printer.Print(cmd.OutOrStdout(), scanResult, printer.FormatJSON)
		if err != nil {
			return err
		}

		return nil
	}
}

func ReturnSuccessfulResponseMock() *grpcs.ScanResult {
	return &grpcs.ScanResult{
		RequestID: "1234567890",
		Status:    true,
		Message:   "Scan completed successfully.",
		ScanDetails: []grpcs.ScanDetail{
			{
				Language:    "Python",
				RuleName:    "Stored XSS",
				Severity:    "High",
				FileName:    "python-vul-file.py",
				Line:        37,
				Remediation: "Fully encode all dynamic data, regardless of source, before embedding it in output.",
				Description: "The method undefined embeds untrusted data in generated output with write, at line 80 of /python-vul-file.py." +
					"This untrusted data is embedded into the output without proper sanitization or encoding, enabling an attacker to inject malicious code into the generated web-page." +
					"The attacker would be able to alter the returned web page by saving malicious data in a data-store ahead of time." +
					"The attacker's modified data is then read from the database by the undefined method with read, at line 37 of /python-vul-file.py." +
					"This untrusted data then flows through the code straight to the output web page, without sanitization.  This can enable a Stored Cross-Site Scripting (XSS) attack.",
			},
			{
				Language:    "Python",
				RuleName:    "Missing HSTS Header",
				Severity:    "Medium",
				FileName:    "python-vul-file.py",
				Line:        76,
				Remediation: "Before setting the HSTS header - consider the implications it may have: Forcing HTTPS will prevent any future use of HTTP",
				Description: "The web-application does not define an HSTS header, leaving it vulnerable to attack.",
			},
		},
	}
}

func ReturnFailureResponseMock() *grpcs.ScanResult {
	return &grpcs.ScanResult{
		RequestID: "some-request-id",
		Status:    false,
		Message:   "Scan failed.",
		Error:     &grpcs.Error{InternalError, "An internal error occurred."},
	}
}

const (
	InternalError = 2
)
