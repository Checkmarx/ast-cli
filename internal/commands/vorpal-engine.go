package commands

import (
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func runScanVorpalCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		vorpalLatestVersion, _ := cmd.Flags().GetBool(commonParams.VorpalLatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)

		scanResult, err := ExecuteVorpalScan(fileSourceFlag, vorpalLatestVersion)
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

func ExecuteVorpalScan(fileSourceFlag string, vorpalUpdateVersion bool) (*ScanResult, error) {
	if fileSourceFlag == "" {
		return nil, nil
	}

	if filepath.Ext(fileSourceFlag) == "" {
		return nil, errors.New(errorConstants.FileExtensionIsRequired)
	}

	//TODO: add vorpal scan logic here
	if vorpalUpdateVersion {
		return ReturnSuccessfulResponseMock(), nil
	}
	//TODO: returning mock failure just to test the vorpalUpdateVersion flag for now
	return ReturnFailureResponseMock(), nil
}

func ReturnSuccessfulResponseMock() *ScanResult {
	return &ScanResult{
		RequestID: "1234567890",
		Status:    true,
		Message:   "Scan completed successfully.",
		ScanDetails: []ScanDetail{
			{
				Language:    "Python",
				QueryName:   "Stored XSS",
				Severity:    "High",
				FileName:    "python-vul-file.py",
				Line:        37,
				Remediation: "Fully encode all dynamic data, regardless of source, before embedding it in output.",
				Description: "The method undefined embeds untrusted data in generated output with write, at line 80 of /dsvw.py. This untrusted data is embedded into the output without proper sanitization or encoding, enabling an attacker to inject malicious code into the generated web-page. The attacker would be able to alter the returned web page by saving malicious data in a data-store ahead of time. The attacker's modified data is then read from the database by the undefined method with read, at line 37 of /dsvw.py. This untrusted data then flows through the code straight to the output web page, without sanitization.  This can enable a Stored Cross-Site Scripting (XSS) attack.",
			},
			{
				Language:    "Python",
				QueryName:   "Missing HSTS Header",
				Severity:    "Medium",
				FileName:    "python-vul-file.py",
				Line:        76,
				Remediation: "Before setting the HSTS header - consider the implications it may have: Forcing HTTPS will prevent any future use of HTTP",
				Description: "The web-application does not define an HSTS header, leaving it vulnerable to attack.",
			},
		},
	}
}

func ReturnFailureResponseMock() *ScanResult {
	return &ScanResult{
		RequestID: "some-request-id",
		Status:    false,
		Message:   "Scan failed.",
		Error:     &Error{InternalError, "An internal error occurred."},
	}
}

type ScanResult struct {
	RequestID   string       `json:"request_id"`
	Status      bool         `json:"status"`
	Message     string       `json:"message"`
	ScanDetails []ScanDetail `json:"scan_details"`
	Error       *Error       `json:"error"`
}

type ScanDetail struct {
	RuleID      uint32 `json:"rule_id"`
	Language    string `json:"language"`
	QueryName   string `json:"query_name"`
	Severity    string `json:"severity"`
	FileName    string `json:"file_name"`
	Line        uint32 `json:"line"`
	Length      uint32 `json:"length"`
	Remediation string `json:"remediation"`
	Description string `json:"description"`
}

type Error struct {
	Code        ErrorCode `json:"code"`
	Description string    `json:"description"`
}

type ErrorCode int32

const (
	UnknownError   = 0
	InvalidRequest = 1
	InternalError  = 2
)
