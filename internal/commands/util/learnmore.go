package util

// nolint:goimports
import (
	"html"
	"log"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const defaultFormat = "list"

type sampleObjectView struct {
	ProgLanguage string `json:"progLanguage"`
	Code         string `json:"code"`
	Title        string `json:"title"`
}

type LearnMoreResponseView struct {
	QueryID                string             `json:"queryId"`
	QueryName              string             `json:"queryName"`
	QueryDescriptionID     string             `json:"queryDescriptionId"`
	ResultDescription      string             `json:"resultDescription"`
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
				https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-815a1110-31ef-7cfb-e640-755fab4fae0d
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
				invalidFlag, params.QueryIDFlag,
			)
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
			format, _ := cmd.Flags().GetString(params.FormatFlag)
			learnMoreResponseView := toLearnMoreResponseView(LearnMoreResponse)
			if format == "" {
				format = defaultFormat
			}
			err := printer.Print(cmd.OutOrStdout(), learnMoreResponseView, format)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toLearnMoreResponseView(response *[]*wrappers.LearnMoreResponse) interface{} {
	var learnMoreResponseView []*LearnMoreResponseView
	for _, resp := range *response {
		learnMoreResponseView = append(
			learnMoreResponseView, &LearnMoreResponseView{
				QueryID:                resp.QueryID,
				QueryName:              resp.QueryName,
				QueryDescriptionID:     resp.QueryDescriptionID,
				ResultDescription:      resp.ResultDescription,
				Risk:                   html.EscapeString(resp.Risk),
				Cause:                  html.EscapeString(resp.Cause),
				GeneralRecommendations: html.EscapeString(resp.GeneralRecommendations),
				Samples:                addSampleResponses(resp.Samples),
			},
		)
	}
	return learnMoreResponseView
}

func addSampleResponses(samples []wrappers.SampleObject) []sampleObjectView {
	var sampleObjectViews []sampleObjectView
	for _, sample := range samples {
		sampleObjectViews = append(
			sampleObjectViews, sampleObjectView{
				ProgLanguage: sample.ProgLanguage,
				Code:         sample.Code,
				Title:        sample.Title,
			},
		)
	}
	return sampleObjectViews
}
