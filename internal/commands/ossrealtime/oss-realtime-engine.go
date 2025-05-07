package ossrealtime

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

var mockPackages = []Response{
	{
		PackageManager: "npm",
		PackageName:    "@ant-design/icons",
		Version:        "2.1.1",
		FilePath:       "package.json",
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
		FilePath:       "package.json",
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
		FilePath:       "package.json",
		LineStart:      30,
		LineEnd:        30,
		StartIndex:     5,
		EndIndex:       20,
		Status:         "Unknown",
	},
}

func RunScanOssRealtimeCommand(jwtWrapper wrappers.JWTWrapper, featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		//fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		// agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)

		packages := buildMockScanResult()

		err := printer.Print(cmd.OutOrStdout(), packages, printer.FormatJSON)
		if err != nil {
			return err
		}

		return nil
	}
}

func buildMockScanResult() []Response {
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
