package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/spf13/viper"

	"github.com/pkg/errors"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	failedDeletingProj    = "Failed deleting a project"
	failedGettingBranches = "Failed getting branches for project"
	projOriginLevel       = "Project"
	repoConfKey           = "scan.handler.git.repository"
	sshConfKey            = "scan.handler.git.sshKey"
	mandatoryRepoURLError = "flag --repo-url is mandatory when --ssh-key is provided"
	invalidRepoURL        = "provided repository url doesn't need a key. Make sure you are defining the right repository or remove the flag --ssh-key"
	emptyTag              = "NONE"
)

var (
	filterProjectsListFlagUsage = fmt.Sprintf(
		"Filter the list of projects. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.IDQueryParam,
				commonParams.IDsQueryParam,
				commonParams.IDRegexQueryParam,
				commonParams.TagsKeyQueryParam,
				commonParams.TagsValueQueryParam,
			}, ",",
		),
	)
	filterBranchesFlagUsage = fmt.Sprintf(
		"Filter the list of branches. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.BranchNameQueryParam,
			}, ",",
		),
	)
)

func NewProjectCommand(applicationsWrapper wrappers.ApplicationsWrapper, projectsWrapper wrappers.ProjectsWrapper, groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	projCmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long:  "The project command enables the ability to manage projects in Checkmarx One.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html
			`,
			),
		},
	}

	createProjCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new project",
		Long:  "The project create command enables the ability to create a new project in Checkmarx One.",
		Example: heredoc.Doc(
			`
			$ cx project create --project-name <Project Name>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html#UUID-44ecd672-8f1f-32de-6c2e-838b680a0bf4
			`,
			),
		},
		RunE: runCreateProjectCommand(applicationsWrapper, projectsWrapper, groupsWrapper, accessManagementWrapper, featureFlagsWrapper),
	}
	createProjCmd.PersistentFlags().String(commonParams.TagList, "", "List of tags, ex: (tagA,tagB:val,etc)")
	createProjCmd.PersistentFlags().String(commonParams.GroupList, "", "List of groups, ex: (PowerUsers,etc)")
	createProjCmd.PersistentFlags().StringP(commonParams.ProjectName, "", "", "Name of project")
	createProjCmd.PersistentFlags().StringP(commonParams.MainBranchFlag, "", "", "Main branch")
	createProjCmd.PersistentFlags().String(commonParams.SSHKeyFlag, "", "Path to ssh private key")
	createProjCmd.PersistentFlags().String(commonParams.RepoURLFlag, "", "Repository URL")
	createProjCmd.PersistentFlags().String(commonParams.ApplicationName, "", "Name of the application to assign with the project")

	listProjectsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects in the system",
		Example: heredoc.Doc(
			`
			$ cx project list --format list
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html#UUID-bd2c6c68-081a-e134-b16b-067aba3a8eae
			`,
			),
		},
		RunE: runListProjectsCommand(projectsWrapper),
	}
	listProjectsCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterProjectsListFlagUsage)

	showProjectCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a project",
		Example: heredoc.Doc(
			`
			$ cx project show --project-id <project_id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html#UUID-a5d021d1-2917-4327-a889-b4f1a9d19b6d
			`,
			),
		},
		RunE: runGetProjectByIDCommand(projectsWrapper),
	}
	addProjectIDFlag(showProjectCmd, "Project ID to show")

	projectBranchesCmd := &cobra.Command{
		Use:   "branches",
		Short: "Show list of branches from a project",
		Example: heredoc.Doc(
			`
			$ cx project branches --project-id <project_id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html
			`,
			),
		},
		RunE: runGetBranchesByIDCommand(projectsWrapper),
	}
	addProjectIDFlag(projectBranchesCmd, "Project ID to get branches")
	projectBranchesCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterBranchesFlagUsage)

	deleteProjCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project",
		Example: heredoc.Doc(
			`
			$ cx project delete --project-id <project_id>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html#UUID-2382b35f-fac9-f169-711b-73570278adb1
			`,
			),
		},
		RunE: runDeleteProjectCommand(projectsWrapper),
	}
	addProjectIDFlag(deleteProjCmd, "Project ID to delete")

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags",
		Example: heredoc.Doc(
			`
			$ cx project tags
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-project.html#UUID-eab37623-899c-e97d-e702-6b1946592986
			`,
			),
		},
		RunE: runGetProjectsTagsCommand(projectsWrapper),
	}

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{showProjectCmd, listProjectsCmd, createProjCmd},
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatList,
	)
	projCmd.AddCommand(createProjCmd, projectBranchesCmd, showProjectCmd, listProjectsCmd, deleteProjCmd, tagsCmd)
	return projCmd
}

func updateProjectRequestValues(input *[]byte, cmd *cobra.Command) error {
	var info map[string]interface{}
	projectName, _ := cmd.Flags().GetString(commonParams.ProjectName)
	mainBranch, _ := cmd.Flags().GetString(commonParams.MainBranchFlag)
	_ = json.Unmarshal(*input, &info)
	if projectName == "" {
		return errors.Errorf(errorConstants.ProjectNameIsRequired)
	}

	info["name"] = projectName

	if mainBranch != "" {
		info["mainBranch"] = mainBranch
	}
	*input, _ = json.Marshal(info)
	return nil
}

func runCreateProjectCommand(
	applicationsWrapper wrappers.ApplicationsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		applicationName, _ := cmd.Flags().GetString(commonParams.ApplicationName)
		var applicationID []string
		if applicationName != "" {
			application, getAppErr := services.GetApplication(applicationName, applicationsWrapper)
			if getAppErr != nil {
				return getAppErr
			}
			if application == nil {
				return errors.Errorf(errorConstants.ApplicationDoesntExistOrNoPermission)
			}
			applicationID = []string{application.ID}
		}

		var input = []byte("{}")
		err := updateProjectRequestValues(&input, cmd)
		if err != nil {
			return err
		}
		groups, err := updateGroupValues(&input, cmd, groupsWrapper)
		if err != nil {
			return err
		}
		// Validate groups access before creating the project.
		// This validation will only be performed if the ACCESS_MANAGEMENT_PHASE2 flag is ON.
		err = services.ValidateGroupsAccessPhase2(groups, accessManagementWrapper, featureFlagsWrapper)
		if err != nil {
			return err
		}

		setupScanTags(&input, cmd)
		err = validateConfiguration(cmd)
		if err != nil {
			return err
		}
		var projModel = wrappers.Project{}
		projModel.ApplicationIds = applicationID
		var projResponseModel *wrappers.ProjectResponseModel
		var errorModel *wrappers.ErrorModel
		// Try to parse to a project model in order to manipulate the request payload
		err = json.Unmarshal(input, &projModel)
		if err != nil {
			return errors.Wrapf(err, "%s: Input in bad format", services.FailedCreatingProj)
		}
		var payload []byte
		payload, _ = json.Marshal(projModel)
		logger.PrintIfVerbose(fmt.Sprintf("Payload to projects service: %s\n", string(payload)))
		projResponseModel, errorModel, err = projectsWrapper.Create(&projModel)
		if err != nil {
			return errors.Wrapf(err, "%s", services.FailedCreatingProj)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, services.FailedCreatingProj, errorModel.Code, errorModel.Message)
		} else if projResponseModel != nil {
			err = printByFormat(cmd, toProjectView(*projResponseModel))
			if err != nil {
				return errors.Wrapf(err, "%s", services.FailedCreatingProj)
			}
		}
		err = services.AssignGroupsToProjectNewAccessManagement(projResponseModel.ID, projResponseModel.Name, groups, accessManagementWrapper, featureFlagsWrapper)
		if err != nil {
			return err
		}
		err = updateProjectConfigurationIfNeeded(cmd, projectsWrapper, projResponseModel.ID)
		if err != nil {
			return err
		}

		return nil
	}
}

func updateProjectConfigurationIfNeeded(cmd *cobra.Command, projectsWrapper wrappers.ProjectsWrapper, projectID string) error {
	// Just update project configuration id a repository url is defined
	if cmd.Flags().Changed(commonParams.RepoURLFlag) {
		var projectConfigurations []wrappers.ProjectConfiguration

		repoURL, _ := cmd.Flags().GetString(commonParams.RepoURLFlag)

		urlConf := getProjectConfiguration(repoConfKey, "repository", git, projOriginLevel, repoURL, "String", true)

		projectConfigurations = append(projectConfigurations, urlConf)

		if cmd.Flags().Changed(commonParams.SSHKeyFlag) {
			sshKeyPath, _ := cmd.Flags().GetString(commonParams.SSHKeyFlag)

			sshKey, sshErr := util.ReadFileAsString(sshKeyPath)
			if sshErr != nil {
				return sshErr
			}

			viper.Set(commonParams.SSHValue, sshKey)

			sshKeyConf := getProjectConfiguration(sshConfKey, "sshKey", git, projOriginLevel, sshKey, "Secret", true)

			projectConfigurations = append(projectConfigurations, sshKeyConf)
		}

		_, configErr := projectsWrapper.UpdateConfiguration(projectID, projectConfigurations)
		if configErr != nil {
			return configErr
		}
	}

	return nil
}

func getProjectConfiguration(key, name, category, level, value, valueType string, allowOverride bool) wrappers.ProjectConfiguration {
	config := wrappers.ProjectConfiguration{}
	config.Key = key
	config.Name = name
	config.Category = category
	config.OriginLevel = level
	config.Value = value
	config.ValueType = valueType
	config.AllowOverride = allowOverride

	return config
}

func validateConfiguration(cmd *cobra.Command) error {
	var sshKeyDefined bool
	var repoURLDefined bool

	// Validate if ssh key is empty when provided
	if cmd.Flags().Changed(commonParams.SSHKeyFlag) {
		sshKey, _ := cmd.Flags().GetString(commonParams.SSHKeyFlag)

		if strings.TrimSpace(sshKey) == "" {
			return errors.New("flag needs an argument: --ssh-key")
		}

		sshKeyDefined = true
	}

	// Validate if repo url is empty when provided
	if cmd.Flags().Changed(commonParams.RepoURLFlag) {
		repoURL, _ := cmd.Flags().GetString(commonParams.RepoURLFlag)

		if strings.TrimSpace(repoURL) == "" {
			return errors.New("flag needs an argument: --repo-url")
		}

		repoURLDefined = true
	}

	// If ssh key is defined we have two checks to validate:
	// 		1. repo url needs to be provided
	// 		2. provided repo url needs to be a ssh url
	if sshKeyDefined {
		if !repoURLDefined {
			return errors.New(mandatoryRepoURLError)
		}

		repoURL, _ := cmd.Flags().GetString(commonParams.RepoURLFlag)

		if !util.IsSSHURL(repoURL) {
			return errors.New(invalidRepoURL)
		}
	}

	return nil
}

func runListProjectsCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allProjectsModel *wrappers.ProjectsCollectionResponseModel
		var errorModel *wrappers.ErrorModel

		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingAll)
		}

		supportEmptyTags(params)

		allProjectsModel, errorModel, err = projectsWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allProjectsModel != nil && allProjectsModel.Projects != nil {
			err = printByFormat(cmd, toProjectViews(allProjectsModel.Projects))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func supportEmptyTags(params map[string]string) {
	if hasNoneKeyAndValue(params) {
		addEmptyTagsParam(params)
	}
}

func hasNoneKeyAndValue(params map[string]string) bool {
	hasNoneKey := hasNoneValueInAttribute(params, commonParams.TagsKeyQueryParam)
	hasNoneValue := hasNoneValueInAttribute(params, commonParams.TagsValueQueryParam)
	return hasNoneKey && hasNoneValue
}

func hasNoneValueInAttribute(params map[string]string, attribute string) bool {
	values, exists := params[attribute]
	return exists && strings.Contains(values, emptyTag)
}

func addEmptyTagsParam(params map[string]string) {
	removeNoneKeyAndValue(params)
	params[commonParams.TagsEmptyQueryParam] = "true"
}

func removeNoneKeyAndValue(params map[string]string) {
	removeNoneAttribute(params, commonParams.TagsKeyQueryParam)
	removeNoneAttribute(params, commonParams.TagsValueQueryParam)
}

func removeNoneAttribute(params map[string]string, attribute string) {
	values, exists := params[attribute]
	if exists {
		values = strings.ReplaceAll(values, ","+emptyTag, "")
		values = strings.ReplaceAll(values, emptyTag+",", "")
		values = strings.ReplaceAll(values, emptyTag, "")
		if values == "" {
			delete(params, attribute)
		} else {
			params[attribute] = values
		}
	}
}

func runGetProjectByIDCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var projectResponseModel *wrappers.ProjectResponseModel
		var errorModel *wrappers.ErrorModel
		var err error
		projectID, _ := cmd.Flags().GetString(commonParams.ProjectIDFlag)
		if projectID == "" {
			return errors.Errorf("%s: Please provide a project ID", services.FailedGettingProj)
		}
		projectResponseModel, errorModel, err = projectsWrapper.GetByID(projectID)
		if err != nil {
			return errors.Wrapf(err, "%s", services.FailedGettingProj)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", services.FailedGettingProj, errorModel.Code, errorModel.Message)
		} else if projectResponseModel != nil {
			resp := GetProjectByName(projectResponseModel.Name, projectsWrapper)

			projectResponseModel.Groups = resp.Groups
			err = printByFormat(cmd, toProjectView(*projectResponseModel))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func GetProjectByName(projectName string, projectsWrapper wrappers.ProjectsWrapper) wrappers.ProjectResponseModel {
	resp, err := services.GetProjectsCollectionByProjectName(projectName, projectsWrapper)
	if err != nil {
		return wrappers.ProjectResponseModel{}
	}

	for i := range resp.Projects {
		project := &resp.Projects[i]
		if project.Name == projectName {
			return *project
		}
	}
	return wrappers.ProjectResponseModel{}
}

func runGetBranchesByIDCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var branches []string
		var errorModel *wrappers.ErrorModel
		var err error

		projectID, _ := cmd.Flags().GetString(commonParams.ProjectIDFlag)
		if projectID == "" {
			return errors.Errorf("%s: Please provide a project ID", failedGettingBranches)
		}
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingAll)
		}

		branches, errorModel, err = projectsWrapper.GetBranchesByID(projectID, params)

		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingBranches)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingBranches, errorModel.Code, errorModel.Message)
		}

		if branches == nil {
			branches = []string{}
		}

		branchesJSON, err := json.Marshal(branches)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to serialize project branches response ", failedGettingBranches)
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(branchesJSON))

		return nil
	}
}

func runDeleteProjectCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var errorModel *wrappers.ErrorModel
		var err error
		projectID, _ := cmd.Flags().GetString(commonParams.ProjectIDFlag)
		if projectID == "" {
			return errors.Errorf("%s: Please provide a project ID", failedDeletingProj)
		}
		errorModel, err = projectsWrapper.Delete(projectID)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedDeletingProj)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedDeletingProj, errorModel.Code, errorModel.Message)
		}
		return nil
	}
}

func runGetProjectsTagsCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var tags map[string][]string
		var errorModel *wrappers.ErrorModel
		var err error

		tags, errorModel, err = projectsWrapper.Tags()
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingTags)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingTags, errorModel.Code, errorModel.Message)
		} else if tags != nil {
			var tagsJSON []byte
			tagsJSON, err = json.Marshal(tags)
			if err != nil {
				return errors.Wrapf(err, "%s: failed to serialize project tags response ", failedGettingTags)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(tagsJSON))
		}
		return nil
	}
}

func toProjectViews(models []wrappers.ProjectResponseModel) []projectView {
	result := make([]projectView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toProjectView(models[i])
	}
	return result
}

func toProjectView(model wrappers.ProjectResponseModel) projectView { //nolint:gocritic
	return projectView{
		ID:             model.ID,
		Name:           model.Name,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
		Tags:           model.Tags,
		Groups:         model.Groups,
		ApplicationIds: model.ApplicationIds,
	}
}

type projectView struct {
	ID             string `format:"name:Project ID"`
	Name           string
	CreatedAt      time.Time `format:"name:Created at;time:01-02-06 15:04:05"`
	UpdatedAt      time.Time `format:"name:Updated at;time:01-02-06 15:04:05"`
	Tags           map[string]string
	Groups         []string
	ApplicationIds []string
}
