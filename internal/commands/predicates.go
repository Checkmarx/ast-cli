package commands

import (
	"encoding/json"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

var systemStates = []wrappers.CustomState{
	{ID: -1, Name: params.ToVerify, Type: ""},
	{ID: -1, Name: params.NotExploitable, Type: ""},
	{ID: -1, Name: params.ProposedNotExploitable, Type: ""},
	{ID: -1, Name: params.CONFIRMED, Type: ""},
	{ID: -1, Name: params.URGENT, Type: ""},
}

func NewResultsPredicatesCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, customStatesWrapper wrappers.CustomStatesWrapper) *cobra.Command {
	triageCmd := &cobra.Command{
		Use:   "triage",
		Short: "Manage results",
		Long:  "The 'triage' command enables the ability to manage results in Checkmarx One",
	}
	triageShowCmd := triageShowSubCommand(resultsPredicatesWrapper)
	triageUpdateCmd := triageUpdateSubCommand(resultsPredicatesWrapper, featureFlagsWrapper, customStatesWrapper)
	triageGetStatesCmd := triageGetStatesSubCommand(customStatesWrapper, featureFlagsWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{triageShowCmd},
		printer.FormatList, printer.FormatTable, printer.FormatJSON,
	)

	triageCmd.AddCommand(triageShowCmd, triageUpdateCmd, triageGetStatesCmd)
	return triageCmd
}

func triageGetStatesSubCommand(customStatesWrapper wrappers.CustomStatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	triageGetStatesCmd := &cobra.Command{
		Use:   "get-states",
		Short: "Show the custom states that have been configured in your tenant",
		Long:  "The get-states command shows information about each of the custom states that have been configured in your tenant account",
		Example: heredoc.Doc(
			`
            $ cx triage get-states
            $ cx triage get-states --all
        `,
		),
		RunE: runTriageGetStates(customStatesWrapper, featureFlagsWrapper),
	}

	triageGetStatesCmd.PersistentFlags().Bool(params.AllStatesFlag, false, "Show all custom states, including the ones that have been deleted")

	return triageGetStatesCmd
}

func runTriageGetStates(customStatesWrapper wrappers.CustomStatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.CustomStatesFeatureFlag)
		if !flagResponse.Status {
			return printer.Print(cmd.OutOrStdout(), systemStates, printer.FormatJSON)
		}
		includeDeleted, _ := cmd.Flags().GetBool(params.AllStatesFlag)
		states, err := customStatesWrapper.GetAllCustomStates(includeDeleted)
		if err != nil {
			return errors.Wrap(err, "Failed to fetch custom states")
		}
		states = append(states, systemStates...)
		err = printer.Print(cmd.OutOrStdout(), states, printer.FormatJSON)
		return err
	}
}

func triageShowSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper) *cobra.Command {
	triageShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Get the predicates history for the given issue",
		Long:  "The show command provides a list of all the predicates in the issue",
		Example: heredoc.Doc(
			`
			$ cx triage show --similarity-id <SimilarityID> --project-id <ProjectID> --scan-type <SAST||IAC-SECURITY||SCS>
		`,
		),

		RunE: runTriageShow(resultsPredicatesWrapper),
	}

	triageShowCmd.PersistentFlags().String(params.SimilarityIDFlag, "", "Similarity ID")
	triageShowCmd.PersistentFlags().String(params.ProjectIDFlag, "", "Project ID")
	triageShowCmd.PersistentFlags().String(params.ScanTypeFlag, "", "Scan Type")

	markFlagAsRequired(triageShowCmd, params.SimilarityIDFlag)
	markFlagAsRequired(triageShowCmd, params.ProjectIDFlag)
	markFlagAsRequired(triageShowCmd, params.ScanTypeFlag)

	return triageShowCmd
}

func triageUpdateSubCommand(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, customStatesWrapper wrappers.CustomStatesWrapper) *cobra.Command {
	triageUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the state, severity or comment for the given issue",
		Long:  "The update command enables the ability to triage the results in Checkmarx One",
		Example: heredoc.Doc(
			`
				$ cx triage update 
				--similarity-id <SimilarityID> 
				--project-id <ProjectID> 
				--state <TO_VERIFY|NOT_EXPLOITABLE|PROPOSED_NOT_EXPLOITABLE|CONFIRMED|URGENT|custom state>
				--state-id <custom state ID>
				--severity <CRITICAL|HIGH|MEDIUM|LOW|INFO> 
				--comment <Comment(Optional)> 
				--scan-type <SAST|IAC-SECURITY>
		`,
		),

		RunE: runTriageUpdate(resultsPredicatesWrapper, featureFlagsWrapper, customStatesWrapper),
	}

	triageUpdateCmd.PersistentFlags().String(params.SimilarityIDFlag, "", "Similarity ID")
	triageUpdateCmd.PersistentFlags().String(params.SeverityFlag, "", "Severity")
	triageUpdateCmd.PersistentFlags().String(params.ProjectIDFlag, "", "Project ID")
	triageUpdateCmd.PersistentFlags().String(params.StateFlag, "", "Specify the state that you would like to apply. Can be a pre-configured state (e.g., not_exploitable) or a custom state created in your account")
	triageUpdateCmd.PersistentFlags().Int(params.CustomStateIDFlag, -1, "Specify the ID of the states that you would like to apply to this result")
	triageUpdateCmd.PersistentFlags().String(params.CommentFlag, "", "Optional comment")
	triageUpdateCmd.PersistentFlags().String(params.ScanTypeFlag, "", "Scan Type")
	triageUpdateCmd.PersistentFlags().StringSlice(params.VulnerabilitiesFlag, []string{}, "List Vulnerabilities string")

	markFlagAsRequired(triageUpdateCmd, params.ProjectIDFlag)
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

func runTriageUpdate(resultsPredicatesWrapper wrappers.ResultsPredicatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, customStatesWrapper wrappers.CustomStatesWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {

		similarityID, _ := cmd.Flags().GetString(params.SimilarityIDFlag)
		projectID, _ := cmd.Flags().GetString(params.ProjectIDFlag)
		severity, _ := cmd.Flags().GetString(params.SeverityFlag)
		state, _ := cmd.Flags().GetString(params.StateFlag)
		customStateID, _ := cmd.Flags().GetInt(params.CustomStateIDFlag)
		comment, _ := cmd.Flags().GetString(params.CommentFlag)
		scanType, _ := cmd.Flags().GetString(params.ScanTypeFlag)
		// check if the current tenant has critical severity available
		flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.CVSSV3Enabled)
		vulnerabilityDetails, _ := cmd.Flags().GetStringSlice(params.VulnerabilitiesFlag)

		criticalEnabled := flagResponse.Status
		if !criticalEnabled && strings.EqualFold(severity, "critical") {
			return errors.Errorf("%s", "Critical severity is not available for your tenant.This severity status will be enabled shortly")
		}
		var err error
		state, customStateID, err = determineSystemOrCustomState(customStatesWrapper, featureFlagsWrapper, state, customStateID)
		predicate, err := handlePredicate(vulnerabilityDetails, similarityID, projectID, severity, state, customStateID, comment, scanType)
		if err != nil {
			return err
		}
		_, err = resultsPredicatesWrapper.PredicateSeverityAndState(predicate, scanType)
		if err != nil {
			return errors.Wrapf(err, "%s", "Failed updating the predicate")
		}
		return nil
	}
}

func handlePredicate(vulnerabilityDetails []string, similarityID string, projectID string, severity string, state string, customStateID int, comment string, scanType string) (interface{}, error) {
	// TODO trim space here
	scanType = strings.ToLower(scanType)
	if strings.EqualFold(scanType, Sca) {
		state = transformState(state)
		payload, err := handleScaTriage(vulnerabilityDetails, comment, state, projectID)
		if err != nil {
			return nil, errors.Errorf("%s", "Failed Sca handling")
		}
		return payload, nil
	} else {
		payload := &wrappers.PredicateRequest{
			SimilarityID: similarityID,
			ProjectID:    projectID,
			Severity:     severity,
			Comment:      comment,
		}
		if state != "" {
			payload.State = &state
		} else {
			payload.CustomStateID = &customStateID
		}
		return payload, nil
	}
}

func transformState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	switch state {
	case strings.ToLower(params.ToVerify):
		return wrappers.ToVerify
	case strings.ToLower(params.URGENT):
		return wrappers.Urgent
	case strings.ToLower(params.NotExploitable):
		return wrappers.NotExploitable
	case strings.ToLower(params.ProposedNotExploitable):
		return wrappers.ProposedNotExploitable
	case strings.ToLower(params.CONFIRMED):
		return wrappers.Confirmed
	}
	return ""
}

// TODO rename the function
func handleScaTriage(vulnerabilityDetails []string, comment string, state string, projectId string) (interface{}, error) {
	scaTriageInfo := make(map[string]interface{})
	for _, vulnerability := range vulnerabilityDetails {
		vulnerabilityKeyVal := strings.SplitN(vulnerability, "=", 2)
		err := validateVulnerabilityDetails(vulnerabilityKeyVal)
		if err != nil {
			return nil, err
		}
		scaTriageInfo[vulnerabilityKeyVal[0]] = vulnerabilityKeyVal[1]
	}
	scaTriageInfo["projectIds"] = []string{projectId}
	actionInfo := make(map[string]interface{})
	actionInfo["actionType"] = params.ChangeState
	actionInfo["value"] = state
	actionInfo["comment"] = comment
	scaTriageInfo["actions"] = []map[string]interface{}{actionInfo}

	b, err := json.Marshal(scaTriageInfo)
	if err != nil {
		return nil, errors.Errorf("Failed to serialize vulnerabilities %s", scaTriageInfo)
	}
	payload := &wrappers.ScaPredicateRequest{}
	err = json.Unmarshal(b, payload)
	if err != nil {
		return nil, errors.Errorf("Failed to deserialize vulnerabilities %s", b)
	}
	return payload, nil
}

func validateVulnerabilityDetails(vulnerability []string) error {
	if len(vulnerability) != params.KeyValuePairSize {
		return errors.Errorf("Invalid vulnerabilities. It should be in a KEY=VALUE format")
	}
	if len(strings.Split(vulnerability[1], ",")) > params.SingleValueSize || len(strings.Split(vulnerability[1], ",")) > params.SingleValueSize {
		return errors.Errorf("Cannot specify multiple values to key %s", vulnerability[0])
	}
	return nil
}

func determineSystemOrCustomState(customStatesWrapper wrappers.CustomStatesWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper, state string, customStateID int) (string, int, error) {
	if !isCustomState(state) {
		return state, -1, nil
	}

	flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.SastCustomStateEnabled)
	sastCustomStateEnabled := flagResponse.Status
	if !sastCustomStateEnabled {
		return "", -1, errors.Errorf("%s", "Custom state is not available for your tenant.")
	}

	if customStateID == -1 {
		if state == "" {
			return "", -1, errors.Errorf("state-id is required when state is not provided")
		}

		var err error
		customStateID, err = getCustomStateID(customStatesWrapper, state)
		if err != nil {
			return "", -1, errors.Wrapf(err, "Failed to get custom state ID for state: %s", state)
		}
	}
	return "", customStateID, nil
}
func isCustomState(state string) bool {
	if state == "" {
		return true
	}
	for _, systemState := range systemStates {
		if strings.EqualFold(state, systemState.Name) {
			return false
		}
	}
	return true
}

func getCustomStateID(customStatesWrapper wrappers.CustomStatesWrapper, state string) (int, error) {
	customStates, err := customStatesWrapper.GetAllCustomStates(false)
	if err != nil {
		return -1, errors.Wrap(err, "Failed to fetch custom states")
	}
	for _, customState := range customStates {
		if customState.Name == state {
			return customState.ID, nil
		}
	}
	return -1, errors.Errorf("No matching state found for %s", state)
}

type predicateView struct {
	ID           string `format:"name:ID"`
	ProjectID    string `format:"name:Project ID"`
	SimilarityID string `format:"name:Similarity ID"`
	Severity     string
	State        string
	StateID      int
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
		StateID:      predicate.StateID,
		Comment:      predicate.Comment,
		CreatedBy:    predicate.CreatedBy,
		CreatedAt:    predicate.CreatedAt,
	}
}
