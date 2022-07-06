package util

import (
	"encoding/json"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/gookit/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
)

const (
	invalidFlag               = "Value of %s is invalid"
	directoryPermission       = 0700
	failedGettingDescriptions = "Failed getting the descriptions"
	defaultFileName           = "cx_descriptions"
	defaultFilePath           = "./"
)

func NewLearnMoreCommand(wrapper wrappers.LearnMoreWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn-more",
		Short: "Shows the descriptions and additional details for a query id",
		Example: heredoc.Doc(
			`
			$ cx utils learn-more
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/wiki/x/VJGXtw
			`,
			),
		},
		RunE: runLearnMoreCmd(wrapper),
	}
	cmd.PersistentFlags().String(params.QueryIDFlag, "", "Query ID is needed")
	cmd.PersistentFlags().String(params.TargetFormatFlag, "", "Default report format is summaryConsole")
	cmd.PersistentFlags().String(params.TargetPathFlag, "", "Default output path is current directory")
	cmd.PersistentFlags().String(params.TargetFlag, "", "Default report name")
	markFlagAsRequired(cmd, params.QueryIDFlag)
	return cmd
}

func runLearnMoreCmd(wrapper wrappers.LearnMoreWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		queryID, _ := cmd.Flags().GetString(params.QueryIDFlag)
		if queryID == "" {
			return errors.Errorf(
				invalidFlag, params.QueryIDFlag)
		}
		pathParams := make(map[string]string)
		pathParams[params.IDsQueryParam] = queryID
		LearnMoreResponse, errorModel, err := wrapper.GetLearnMoreDetails(pathParams)
		if err != nil {
			return err
		}

		if errorModel != nil {
			return errors.Errorf("Failed getting additional details")
		}

		if LearnMoreResponse != nil {
			logger.PrintIfVerbose(fmt.Sprintf("Response from wrapper: %s", LearnMoreResponse))
			createReport(LearnMoreResponse, cmd)
		}

		return nil
	}

}

func createReport(response *[]wrappers.LearnMoreResponseModel, cmd *cobra.Command) error {
	targetFile, _ := cmd.Flags().GetString(params.TargetFlag)
	if targetFile == "" {
		targetFile = defaultFileName
	}
	targetPath, _ := cmd.Flags().GetString(params.TargetPathFlag)
	if targetPath == "" {
		targetPath = defaultFilePath
	}
	format, _ := cmd.Flags().GetString(params.TargetFormatFlag)
	return createDescriptionsReport(targetFile, targetPath, format, response)

}

func createDescriptionsReport(file string, path string, format string, response *[]wrappers.LearnMoreResponseModel) error {
	if path != "" {
		err := createDirectory(path)
		if err != nil {
			return err
		}
	}
	err := writeSummaryConsole(response)
	if err != nil {
		return err
	}
	if printer.IsFormat(format, printer.FormatJSON) {
		jsonRpt := createTargetName(file, path, "json")
		return exportJSONResults(jsonRpt, response)
	}
	return nil
}

func writeSummaryConsole(response *[]wrappers.LearnMoreResponseModel) error {
	for index, resp := range *response {
		color.Bold.Printf("%d) %s:\n", index+1, resp.QueryName)
		color.Bold.Printf("Risk: \n")
		color.Bold.Printf("What might happen? \n")
		color.BgDefault.Printf("%s \n\n", resp.Risk)
		color.Bold.Printf("Cause: \n")
		color.Bold.Printf("How does it happen? \n")
		color.BgDefault.Printf("%s \n\n", resp.Cause)
		color.Bold.Printf("General Recommendations: \n")
		color.Bold.Printf("How to avoid it?")
		color.BgDefault.Printf("\n")
		color.BgDefault.Printf("%s \n\n", resp.GeneralRecommendations)
		color.Bold.Printf("Code samples: \n")
		for sampleIndex, sample := range resp.Samples {
			color.Bold.Printf("%d) %s:\n", sampleIndex+1, sample.Title)
			color.Bold.Printf("Programming Language: %s ", sample.ProgLanguage)
			color.BgDefault.Printf("\n")
			color.BgDefault.Printf("%s \n\n", sample.Code)
		}
	}
	return nil
}

func markFlagAsRequired(cmd *cobra.Command, flag string) {
	err := cmd.MarkPersistentFlagRequired(flag)
	if err != nil {
		log.Fatal(err)
	}
}

func createDirectory(targetPath string) error {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		log.Printf("\nOutput path not found: %s\n", targetPath)
		log.Printf("Creating directory: %s\n", targetPath)
		err = os.Mkdir(targetPath, directoryPermission)
		if err != nil {
			return err
		}
	}
	return nil
}

func createTargetName(targetFile, targetPath, targetType string) string {
	return filepath.Join(targetPath, targetFile+"."+targetType)
}

func exportJSONResults(targetFile string, results *[]wrappers.LearnMoreResponseModel) error {
	var err error
	var resultsJSON []byte
	log.Println("Creating JSON Report: ", targetFile)
	resultsJSON, err = json.Marshal(results)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize descriptions response ", failedGettingDescriptions)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create descriptions target file  ", failedGettingDescriptions)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	_ = f.Close()
	return nil
}
