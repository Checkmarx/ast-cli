package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewResultsPredicatesCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, customStatesWrapper wrappers.CustomStatesWrapper ) *cobra.Command {
	triageCmd := &cobra.Command{
		Use:   "triage",
		Short: "Manage results",
		Long:  "The 'triage' command enables the ability to manage results in Checkmarx One.",
	}
	triageShowCmd := triageShowSubCommand(resultsPredicatesWrapper)
	triageUpdateCmd := triageUpdateSubCommand(resultsPredicatesWrapper, featureFlagsWrapper)
	triageGetStatesCmd := triageGetStatesSubCommand(customStatesWrapper)


	addFormatFlagToMultipleCommands(
		[]*cobra.Command{triageShowCmd},
		printer.FormatList, printer.FormatTable, printer.FormatJSON,
	)

	triageCmd.AddCommand(triageShowCmd, triageUpdateCmd, triageGetStatesCmd)
	return triageCmd
}

func triageGetStatesSubCommand(customStatesWrapper wrappers.CustomStatesWrapper) *cobra.Command {
	triageGetStatesCmd := &cobra.Command{
		Use:   "get-states",
		Short: "Fetch and display custom states.",
		Long:  "Retrieves a list of custom states and prints their names.",
		Example: heredoc.Doc(
			`
            $ cx triage get-states
            $ cx triage get-states --all
        `,
		),
		RunE: runTriageGetStates(customStatesWrapper),
	}

	triageGetStatesCmd.PersistentFlags().Bool(params.AllStatesFlag, false, "Include deleted states")

	return triageGetStatesCmd
}

func runTriageGetStates(customStatesWrapper wrappers.CustomStatesWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		includeDeleted, _ := cmd.Flags().GetBool(params.AllStatesFlag)

		states, err := customStatesWrapper.GetAllCustomStates(includeDeleted)
		if err != nil {
			return errors.Wrap(err, "Failed to fetch custom states")
		}

		for _, state := range states {
			fmt.Println(strings.TrimSpace(state.Name))
		}

		return nil
	}
}


func triageShowSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Get the predicates history for the given issue.",
		Long:  "The show command provides a list of all the predicates in the issue.",
		Example: heredoc.Doc(
			`
			$ cx triage show --similarity-id <SimilarityID> --project-id <ProjectID> --scan-type <SAST||IAC-SECURITY>
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

func triageUpdateSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	triageUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the state, severity or comment for the given issue",
		Long:  "The update command enables the ability to triage the results in Checkmarx One.",
		Example: heredoc.Doc(
			`
				$ cx triage update 
				--similarity-id <SimilarityID> 
				--project-id <ProjectID> 
				--state <TO_VERIFY|NOT_EXPLOITABLE|PROPOSED_NOT_EXPLOITABLE|CONFIRMED|URGENT> 
				--severity <CRITICAL|HIGH|MEDIUM|LOW|INFO> 
				--comment <Comment(Optional)> 
				--scan-type <SAST|IAC-SECURITY>
		`,
		),
		RunE: runTriageUpdate(resultsPredicatesWrapper, featureFlagsWrapper),
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
		var errorModel *wrappers.WebError
		var err error

		similarityID, _ := cmd.Flags().GetString(params.SimilarityIDFlag)
		scanType, _ := cmd.Flags().GetString(params.ScanTypeFlag)
		projectID, _ := cmd.Flags().GetString(params.ProjectIDFlag)

		projectIDs := strings.Split(projectID, ",")
		if len(projectIDs) > 1 {
			return errors.Errorf("%s", "Multiple project-ids are not allowed.")
		}

		predicatesCollection, errorModel, err = resultsPredicatesWrapper.GetAllPredicatesForSimilarityID(
			similarityID,
			projectID,
			scanType,
		)

		if err != nil {
			return errors.Wrapf(err, "%s", "Failed showing the predicate")
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(
				"%s: CODE: %d, %s",
				"Failed showing the predicate.",
				errorModel.Code,
				errorModel.Message,
			)
		} else if predicatesCollection != nil {
			err = printByFormat(cmd, toPredicatesView(*predicatesCollection))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func runTriageUpdate(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		similarityID, _ := cmd.Flags().GetString(params.SimilarityIDFlag)
		projectID, _ := cmd.Flags().GetString(params.ProjectIDFlag)
		severity, _ := cmd.Flags().GetString(params.SeverityFlag)
		state, _ := cmd.Flags().GetString(params.StateFlag)
		comment, _ := cmd.Flags().GetString(params.CommentFlag)
		scanType, _ := cmd.Flags().GetString(params.ScanTypeFlag)
		// check if the current tenant has critical severity available
		flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.CVSSV3Enabled)
		criticalEnabled := flagResponse.Status
		if !criticalEnabled && strings.EqualFold(severity, "critical") {
			return errors.Errorf("%s", "Critical severity is not available for your tenant.This severity status will be enabled shortly")
		}
		predicate := &wrappers.PredicateRequest{
			SimilarityID: similarityID,
			ProjectID:    projectID,
			Severity:     severity,
			State:        state,
			Comment:      comment,
		}

		_, err := resultsPredicatesWrapper.PredicateSeverityAndState(predicate, scanType)
		if err != nil {
			return errors.Wrapf(err, "%s", "Failed updating the predicate")
		}

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
	projectPredicatesCollection := predicatesCollection.PredicateHistoryPerProject

	if len(projectPredicatesCollection) > 0 {
		predicatesPerProject := predicatesCollection.PredicateHistoryPerProject[0]
		predicatesOfSingleProject := predicatesPerProject.Predicates

		views := make([]predicateView, len(predicatesOfSingleProject))
		for i := 0; i < len(predicatesOfSingleProject); i++ {
			views[i] = toSinglePredicateView(&predicatesOfSingleProject[i])
		}

		return views
	}
	views := make([]predicateView, 0)
	return views
}

func toSinglePredicateView(predicate *wrappers.Predicate) predicateView {
	return predicateView{
		ID:           predicate.ID,
		ProjectID:    predicate.ProjectID,
		SimilarityID: predicate.SimilarityID,
		Severity:     predicate.Severity,
		State:        predicate.State,
		Comment:      predicate.Comment,
		CreatedBy:    predicate.CreatedBy,
		CreatedAt:    predicate.CreatedAt,
	}
}
