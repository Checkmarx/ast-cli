package commands

import (
	"path/filepath"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func runScanLightweightCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		engineUpdateVersion, _ := cmd.Flags().GetBool(commonParams.LightweightUpdateVersion)
		sourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)

		scanResult, err := ExecuteLightweightScan(sourceFlag, engineUpdateVersion)
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

func ExecuteLightweightScan(sourceFlag string, _ bool) (*ScanResult, error) {
	if sourceFlag == "" {
		return nil, errors.Errorf(errorConstants.FileSourceFlagIsRequired)
	}

	if filepath.Ext(sourceFlag) == "" {
		return nil, errors.New(errorConstants.FileExtensionIsRequired)
	}

	successfulScan := ReturnSuccessfulResponseMock()
	return successfulScan, nil
}

func ReturnSuccessfulResponseMock() *ScanResult {
	return &ScanResult{
		RequestID: "1234567890",
		Status:    true,
		Message:   "Scan completed successfully.",
		ScanDetails: []ScanDetail{
			{
				Language:    "Java",
				QueryName:   "SQL Injection",
				Severity:    "High",
				FileName:    "Main.java",
				Line:        42,
				Remediation: "Use prepared statements or parameterized queries to prevent SQL injection.",
				Description: "This vulnerability allows an attacker to execute malicious SQL queries.",
			},
			{
				Language:    "JavaScript",
				QueryName:   "Cross-Site Scripting (XSS)",
				Severity:    "Medium",
				FileName:    "index.js",
				Line:        27,
				Remediation: "Escape or sanitize user input before rendering it in HTML to prevent XSS attacks.",
				Description: "This vulnerability allows an attacker to inject malicious scripts into web pages viewed by other users.",
			},
		},
	}
}

type ScanRequest struct {
	ID         string `json:"id"`
	FileName   string `json:"file_name"`
	SourceCode string `json:"source_code"`
}

type SingleScanRequest struct {
	ScanRequest ScanRequest `json:"scan_request"`
}

type ScanResult struct {
	RequestID   string       `json:"request_id"`
	Status      bool         `json:"status"`
	Message     string       `json:"message"`
	ScanDetails []ScanDetail `json:"scan_details"`
	Error       Error        `json:"error"`
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
