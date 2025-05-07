package ossrealtime

import (
	"errors"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanOssRealtimeCommand(jwtWrapper wrappers.JWTWrapper, featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errors.New("file source flag is required")
		}
		// agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)

		packages := buildMockScanResult(fileSourceFlag)

		err := printer.Print(cmd.OutOrStdout(), packages, printer.FormatJSON)
		if err != nil {
			return err
		}

		return nil
	}
}

func buildMockScanResult(fileSource string) []Response {
	mockPackages := []Response{
		{
			PackageManager: "npm",
			PackageName:    "@ant-design/icons",
			Version:        "2.1.1",
			FilePath:       fileSource,
			LineStart:      23,
			LineEnd:        23,
			StartIndex:     5,
			EndIndex:       20,
			Status:         "OK",
		},
		{
			PackageManager: "npm",
			PackageName:    "@babel/cli",
			Version:        "7.12.1",
			FilePath:       fileSource,
			LineStart:      26,
			LineEnd:        26,
			StartIndex:     5,
			EndIndex:       20,
			Status:         "Malicious",
		},
		{
			PackageManager: "npm",
			PackageName:    "express",
			Version:        "4.17.1",
			FilePath:       fileSource,
			LineStart:      30,
			LineEnd:        30,
			StartIndex:     5,
			EndIndex:       20,
			Status:         "Unknown",
		},
	}

	return mockPackages
}

type Response struct {
	PackageManager string `json:"PackageManager"`
	PackageName    string `json:"PackageName"`
	Version        string `json:"Version"`
	FilePath       string `json:"Filepath"` // note lowercase “p” to match your JSON
	LineStart      int    `json:"LineStart"`
	LineEnd        int    `json:"LineEnd"`
	StartIndex     int    `json:"StartIndex"`
	EndIndex       int    `json:"EndIndex"`
	Status         string `json:"Status,omitempty"`
}
