package commands

import (
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	KICS = "kics"
)

func NewResultsPredicatesCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageCmd := &cobra.Command{
		Use:   "triage",
		Short: "Manage results",
		Long:  "The 'triage' command enables the ability to manage results in CxAST.",
	}
	triageShowCmd := triageShowSubCommand(resultsPredicatesWrapper)
	triageUpdateCmd := triageUpdateSubCommand(resultsPredicatesWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{triageShowCmd},
		util.FormatList, util.FormatTable, util.FormatJSON,
	)

	triageCmd.AddCommand(triageShowCmd, triageUpdateCmd)
	return triageCmd

}

func triageShowSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Get the predicates history for the given issue.",
		Long:  "The show command provides a list of all the predicates in the issue.",
		Example: heredoc.Doc(
			`
			$ cx triage show --similarity-id <SimilarityID> --project-id <ProjectID> --scan-type <SAST||KICS>
		`,
		),

		RunE: runTriageShow(resultsPredicatesWrapper),
	}

	triageShowCmd.PersistentFlags().String(params.SimilarityIDFlag, "", "Similarity ID")
	triageShowCmd.PersistentFlags().String(params.ProjectIDFlag, "", "Project ID.")
	triageShowCmd.PersistentFlags().String(params.ScanTypeFlag, "", "Scan Type")

	markFlagAsRequired(triageShowCmd, params.SimilarityIDFlag)
	markFlagAsRequired(triageShowCmd, params.ProjectIDFlag)
	markFlagAsRequired(triageShowCmd, params.ScanTypeFlag)

	return triageShowCmd
}

func triageUpdateSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the state, severity or comment for the given issue",
		Long:  "The update command enables the ability to triage the results in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx triage update --similarity-id <SimilarityID> --project-id <ProjectID> --state <TO_VERIFY, NOT_EXPLOITABLE, PROPOSED_NOT_EXPLOITABLE, CONFIRMED, URGENT> --severity <HIGH, MEDIUM, LOW, INFO > --comment <Comment(Optional)>--scan-type <SAST||KICS>
		`,
		),
		RunE: runTriageUpdate(resultsPredicatesWrapper),
	}

	triageUpdateCmd.PersistentFlags().String(params.SimilarityIDFlag, "", "Similarity ID")
	triageUpdateCmd.PersistentFlags().String(params.SeverityFlag, "", "Severity")
	triageUpdateCmd.PersistentFlags().String(params.ProjectIDFlag, "", "Project ID.")
	triageUpdateCmd.PersistentFlags().String(params.StateFlag, "", "State")
	triageUpdateCmd.PersistentFlags().String(params.CommentFlag, "", "Optional comment.")
	triageUpdateCmd.PersistentFlags().String(params.ScanTypeFlag, "", "Scan Type")

	markFlagAsRequired(triageUpdateCmd, params.SimilarityIDFlag)
	markFlagAsRequired(triageUpdateCmd, params.SeverityFlag)
	markFlagAsRequired(triageUpdateCmd, params.ProjectIDFlag)
	markFlagAsRequired(triageUpdateCmd, params.StateFlag)
	markFlagAsRequired(triageUpdateCmd, params.ScanTypeFlag)

	return triageUpdateCmd

}

func runTriageShow(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {

		var predicatesCollection *wrappers.PredicatesCollectionResponseModel
		var errorModel *resultsHelpers.WebError
		var err error

		similarityID, _ := cmd.Flags().GetString(params.SimilarityIDFlag)
		scanType, _ := cmd.Flags().GetString(params.ScanTypeFlag)
		projectId, _ := cmd.Flags().GetString(params.ProjectIDFlag)

		projectIDs := strings.Split(projectId, ",")
		if len(projectIDs) > 1 {
			return errors.Errorf("%s", "Multiple project-ids are not allowed.")
		}

		predicatesCollection, errorModel, err = resultsPredicatesWrapper.GetAllPredicatesForSimilarityID(similarityID, projectId, scanType)

		if err != nil {
			return errors.Wrapf(err, "%s", "Failed getting the predicate.")
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", "Failed getting the predicate.", errorModel.Code, errorModel.Message)
		} else if predicatesCollection != nil {
			err = printByFormat(cmd, toPredicatesView(*predicatesCollection))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func runTriageUpdate(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		similarityID, _ := cmd.Flags().GetString(params.SimilarityIDFlag)
		projectID, _ := cmd.Flags().GetString(params.ProjectIDFlag)
		severity, _ := cmd.Flags().GetString(params.SeverityFlag)
		state, _ := cmd.Flags().GetString(params.StateFlag)
		comment, _ := cmd.Flags().GetString(params.CommentFlag)
		scanType, _ := cmd.Flags().GetString(params.ScanTypeFlag)

		if strings.EqualFold(strings.TrimSpace(scanType), KICS) {
			scanType = "" // API Endpoint for Kics doesnt require scantype.
		}

		predicate := &wrappers.PredicateRequest{
			SimilarityID: similarityID,
			ProjectID:    projectID,
			Severity:     severity,
			State:        state,
			Comment:      comment,
			ScannerType:  scanType,
		}

		_, _ = resultsPredicatesWrapper.PredicateSeverityAndState(predicate)

		return nil
	}
}

type predicateView struct {
	ID           string `format:"name:ID"`
	ProjectID    string `format:"name:Project ID"`
	SimilarityID string `format:"name:Similarity ID"`
	Severity     string
	State        string
	Comment      string
	CreatedBy    string
	CreatedAt    time.Time `format:"name:Created at;time:01-02-06 15:04:05"`
}

func toPredicatesView(predicatesCollection wrappers.PredicatesCollectionResponseModel) []predicateView {
	predicatesPerProject := predicatesCollection.PredicateHistoryPerProject[0]
	predicatesOfSingleProject := predicatesPerProject.Predicates

	views := make([]predicateView, len(predicatesOfSingleProject))
	for i := 0; i < len(predicatesOfSingleProject); i++ {
		views[i] = toSinglePredicateView(predicatesOfSingleProject[i])
	}
	return views
}

func toSinglePredicateView(predicate wrappers.Predicate) predicateView {

	return predicateView{
		ID:           predicate.Id,
		ProjectID:    predicate.ProjectID,
		SimilarityID: predicate.SimilarityID,
		Severity:     predicate.Severity,
		State:        predicate.State,
		Comment:      predicate.Comment,
		CreatedBy:    predicate.CreatedBy,
		CreatedAt:    predicate.CreatedAt,
	}
}
