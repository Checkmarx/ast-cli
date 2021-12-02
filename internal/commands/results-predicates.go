package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewResultsPredicatesCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageCmd := &cobra.Command{
		Use:   "triage",
		Short: "Allows to modify the status and severity of a vulnerability.",
		Example: heredoc.Doc(`
			$ cx scan logs --scan-id <scan Id> --scan-type <sast | sca | kics>
		`),
		RunE: runTriage(resultsPredicatesWrapper),
	}
	triageCmd.PersistentFlags().String(params.SimilarityIdFlag, "", "Similarity ID")
	triageCmd.PersistentFlags().String(params.SeverityFlag, "", "Severity")
	triageCmd.PersistentFlags().String(params.ProjectIDFlag, "", "Project ID.")
	triageCmd.PersistentFlags().String(params.StateFlag, "", "State")
	triageCmd.PersistentFlags().String(params.CommentFlag, "Done via CLI." , "Optional comment.")
	triageCmd.PersistentFlags().String(params.ScanTypeFlag, "" , "Scan Type")

	markFlagAsRequired(triageCmd,params.SimilarityIdFlag)
	markFlagAsRequired(triageCmd, params.ProjectIDFlag)

	return triageCmd
}

func runTriage(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		similarityId, _ := cmd.Flags().GetString(params.SimilarityIdFlag)
		projectId, _ := cmd.Flags().GetString(params.ProjectIDFlag)
		severity, _ := cmd.Flags().GetString(params.SeverityFlag)
		state, _ := cmd.Flags().GetString(params.StateFlag)
		comment, _ := cmd.Flags().GetString(params.CommentFlag)
		scantype, _ := cmd.Flags().GetString(params.ScanTypeFlag)


		//var errorModel *resultsHelpers.WebError

		predicate := &wrappers.Predicate{
			SimilarityId:        similarityId,
			ProjectId:       projectId,
			Severity: severity,
			State:      state,
			Comment: comment,
			ScannerType: scantype,
		}

		_ = resultsPredicatesWrapper.PredicateSeverityAndStateForSAST(predicate)

		//if err != nil {
		//	//return nil, errors.Wrapf(err, "%s", "Error while setting the predicates.")
		//}

		return nil
	}
}