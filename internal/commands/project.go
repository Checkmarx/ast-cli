package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"

	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
	"github.com/spf13/cobra"
)

const (
	failedCreatingProj = "Failed creating a project"
	failedGettingProj  = "Failed getting a project"
	failedDeletingProj = "Failed deleting a project"
)

var (
	filterProjectsListFlagUsage = fmt.Sprintf("Filter the list of projects. Use ';' as the delimeter for arrays. Available filters are: %s",
		strings.Join([]string{
			commonParams.LimitQueryParam,
			commonParams.OffsetQueryParam,
			commonParams.IDQueryParam,
			commonParams.IDsQueryParam,
			commonParams.IDRegexQueryParam,
			commonParams.TagsKeyQueryParam,
			commonParams.TagsValueQueryParam}, ","))
)

func NewProjectCommand(projectsWrapper wrappers.ProjectsWrapper) *cobra.Command {
	projCmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	createProjCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new project",
		RunE:  runCreateProjectCommand(projectsWrapper),
	}
	createProjCmd.PersistentFlags().String(TagList, "", "List of tags, ex: (tagA,tagB:val,etc)")
	createProjCmd.PersistentFlags().String(GroupList, "", "List of groups, ex: (PowerUsers,etc)")
	createProjCmd.PersistentFlags().StringP(ProjectName, "", "", "Name of project")
	createProjCmd.PersistentFlags().StringP(MainBranchFlag, "", "", "Main branch")

	listProjectsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects in the system",
		RunE:  runListProjectsCommand(projectsWrapper),
	}
	listProjectsCmd.PersistentFlags().StringSlice(FilterFlag, []string{}, filterProjectsListFlagUsage)

	showProjectCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a project",
		RunE:  runGetProjectByIDCommand(projectsWrapper),
	}
	addProjectIDFlag(showProjectCmd, "Project ID to show.")

	deleteProjCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project",
		RunE:  runDeleteProjectCommand(projectsWrapper),
	}
	addProjectIDFlag(deleteProjCmd, "Project ID to delete.")

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags",
		RunE:  runGetProjectsTagsCommand(projectsWrapper),
	}

	addFormatFlagToMultipleCommands([]*cobra.Command{showProjectCmd, listProjectsCmd, createProjCmd}, FormatTable,
		FormatJSON, FormatList)
	projCmd.AddCommand(createProjCmd, showProjectCmd, listProjectsCmd, deleteProjCmd, tagsCmd)
	return projCmd
}

func updateProjectRequestValues(input *[]byte, cmd *cobra.Command) error {
	var info map[string]interface{}
	projectName, _ := cmd.Flags().GetString(ProjectName)
	mainBranch, _ := cmd.Flags().GetString(MainBranchFlag)
	repoURL, _ := cmd.Flags().GetString(RepoURLFlag)
	_ = json.Unmarshal(*input, &info)
	if projectName != "" {
		info["name"] = projectName
	} else {
		return errors.Errorf("Project name is required")
	}
	if mainBranch != "" {
		info["mainBranch"] = mainBranch
	}
	if repoURL != "" {
		info["repoUrl"] = repoURL
	}
	*input, _ = json.Marshal(info)
	return nil
}

func updateGroupValues(input *[]byte, cmd *cobra.Command) {
	groupListStr, _ := cmd.Flags().GetString(GroupList)
	groups := strings.Split(groupListStr, ",")
	var groupMap []string
	var info map[string]interface{}
	_ = json.Unmarshal(*input, &info)
	if _, ok := info["groups"]; !ok {
		_ = json.Unmarshal([]byte("[]"), &groupMap)
		info["groups"] = groupMap
	}
	for _, group := range groups {
		if len(group) > 0 {
			groupMap = append(groupMap, group)
		}
	}
	info["groups"] = groupMap
	*input, _ = json.Marshal(info)
}

func runCreateProjectCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var input = []byte("{}")
		var err error
		err = updateProjectRequestValues(&input, cmd)
		if err != nil {
			return err
		}
		updateTagValues(&input, cmd)
		updateGroupValues(&input, cmd)
		var projModel = projectsRESTApi.Project{}
		var projResponseModel *projectsRESTApi.ProjectResponseModel
		var errorModel *projectsRESTApi.ErrorModel
		// Try to parse to a project model in order to manipulate the request payload
		err = json.Unmarshal(input, &projModel)
		if err != nil {
			return errors.Wrapf(err, "%s: Input in bad format", failedCreatingProj)
		}
		var payload []byte
		payload, _ = json.Marshal(projModel)
		PrintIfVerbose(fmt.Sprintf("Payload to projects service: %s\n", string(payload)))

		projResponseModel, errorModel, err = projectsWrapper.Create(&projModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedCreatingProj)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedCreatingProj, errorModel.Code, errorModel.Message)
		} else if projResponseModel != nil {
			err = printByFormat(cmd, toProjectView(*projResponseModel))
			if err != nil {
				return errors.Wrapf(err, "%s", failedCreatingProj)
			}
		}
		return nil
	}
}

func runListProjectsCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allProjectsModel *projectsRESTApi.ProjectsCollectionResponseModel
		var errorModel *projectsRESTApi.ErrorModel

		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingAll)
		}

		allProjectsModel, errorModel, err = projectsWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allProjectsModel != nil && allProjectsModel.Projects != nil {
			err = printByFormat(cmd, toProjectViews(allProjectsModel.Projects))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runGetProjectByIDCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var projectResponseModel *projectsRESTApi.ProjectResponseModel
		var errorModel *projectsRESTApi.ErrorModel
		var err error
		projectID, _ := cmd.Flags().GetString(ProjectIDFlag)
		if projectID == "" {
			return errors.Errorf("%s: Please provide a project ID", failedGettingProj)
		}
		projectResponseModel, errorModel, err = projectsWrapper.GetByID(projectID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingProj)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingProj, errorModel.Code, errorModel.Message)
		} else if projectResponseModel != nil {
			err = printByFormat(cmd, toProjectView(*projectResponseModel))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func runDeleteProjectCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var errorModel *projectsRESTApi.ErrorModel
		var err error
		projectID, _ := cmd.Flags().GetString(ProjectIDFlag)
		if projectID == "" {
			return errors.Errorf("%s: Please provide a project ID", failedDeletingProj)
		}
		errorModel, err = projectsWrapper.Delete(projectID)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedDeletingProj)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedDeletingProj, errorModel.Code, errorModel.Message)
		}
		return nil
	}
}

func runGetProjectsTagsCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var tags map[string][]string
		var errorModel *projectsRESTApi.ErrorModel
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
			fmt.Fprintln(cmd.OutOrStdout(), string(tagsJSON))
		}
		return nil
	}
}
func toProjectViews(models []projectsRESTApi.ProjectResponseModel) []projectView {
	result := make([]projectView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toProjectView(models[i])
	}
	return result
}

func toProjectView(model projectsRESTApi.ProjectResponseModel) projectView { //nolint:gocritic
	return projectView{
		ID:        model.ID,
		Name:      model.Name,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
		Tags:      model.Tags,
		Groups:    model.Groups,
	}
}

type projectView struct {
	ID        string `format:"name:Project ID"`
	Name      string
	CreatedAt time.Time `format:"name:Created at;time:01-02-06 15:04:05"`
	UpdatedAt time.Time `format:"name:Updated at;time:01-02-06 15:04:05"`
	Tags      map[string]string
	Groups    []string
}
