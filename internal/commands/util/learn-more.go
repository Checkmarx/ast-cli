package util

import (
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
)

const (
	invalidFlag = "Value of %s is invalid"
)

type sampleObjectView struct {
	ProgLanguage string `json:"progLanguage"`
	Code         string `json:"code"`
	Title        string `json:"title"`
}

type LearnMoreResponseView struct {
	QueryId                string             `json:"queryId"`
	QueryName              string             `json:"queryName"`
	QueryDescriptionId     string             `json:"queryDescriptionId"`
	ResultDescription      string             `json:"resultDescription"'`
	Risk                   string             `json:"risk"`
	Cause                  string             `json:"cause"`
	GeneralRecommendations string             `json:"generalRecommendations"`
	Samples                []sampleObjectView `json:"samples"`
}

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
	cmd.PersistentFlags().String(params.FormatFlag, "", "Output in json/list/table format")
	err := cmd.MarkPersistentFlagRequired(params.QueryIDFlag)
	if err != nil {
		log.Fatal(err)
	}
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
			format, _ := cmd.Flags().GetString(params.FormatFlag)
			if format != "" {
				learnMoreResponseView := toLearnMoreResponseView(LearnMoreResponse)
				err := printer.Print(cmd.OutOrStdout(), learnMoreResponseView, format)
				if err != nil {
					return err
				}
			} else {
				//err := writeSummaryConsole(LearnMoreResponse)
				err := printSummaryConsole(LearnMoreResponse)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

}

func toLearnMoreResponseView(response *[]*wrappers.LearnMoreResponse) interface{} {
	var learnMoreResponseView []*LearnMoreResponseView
	for _, resp := range *response {
		learnMoreResponseView = append(learnMoreResponseView, &LearnMoreResponseView{
			QueryId:                resp.QueryId,
			QueryName:              resp.QueryName,
			QueryDescriptionId:     resp.QueryDescriptionId,
			ResultDescription:      resp.ResultDescription,
			Risk:                   resp.Risk,
			Cause:                  resp.Cause,
			GeneralRecommendations: resp.GeneralRecommendations,
			Samples:                addSampleResponses(resp.Samples),
		})
	}
	return learnMoreResponseView
}

func addSampleResponses(samples []wrappers.SampleObject) []sampleObjectView {
	var sampleObjectViews []sampleObjectView
	for _, sample := range samples {
		sampleObjectViews = append(sampleObjectViews, sampleObjectView{
			ProgLanguage: sample.ProgLanguage,
			Code:         sample.Code,
			Title:        sample.Title,
		})
	}
	return sampleObjectViews
}

func writeSummaryConsole(response *[]*wrappers.LearnMoreResponse) error {
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

func printSummaryConsole(response *[]*wrappers.LearnMoreResponse) error {
	for index, resp := range *response {
		fmt.Printf("%d) %s:\n", index+1, resp.QueryName)
		fmt.Printf("Risk: \n")
		fmt.Printf("What might happen? \n")
		fmt.Printf("%s \n\n", resp.Risk)
		fmt.Printf("Cause: \n")
		fmt.Printf("How does it happen? \n")
		fmt.Printf("%s \n\n", resp.Cause)
		fmt.Printf("General Recommendations: \n")
		fmt.Printf("How to avoid it?")
		fmt.Printf("\n")
		fmt.Printf("%s \n\n", resp.GeneralRecommendations)
		fmt.Printf("Code samples: \n")
		for sampleIndex, sample := range resp.Samples {
			fmt.Printf("%d) %s:\n", sampleIndex+1, sample.Title)
			fmt.Printf("Programming Language: %s ", sample.ProgLanguage)
			fmt.Printf("\n")
			fmt.Printf("%s \n\n", sample.Code)
		}
	}
	return nil
}
