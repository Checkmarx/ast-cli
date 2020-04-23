package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	wrappers "github.com/checkmarxDev/ast-cli/internal/wrappers"
	projectsRESTApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
	"github.com/spf13/cobra"
)

const (
	failedCreatingProj = "Failed creating a project"
	failedGettingProj  = "Failed getting a project"
	failedDeletingProj = "Failed deleting a project"
)

func NewProjectCommand(projectsWrapper wrappers.ProjectsWrapper) *cobra.Command {
	projCmd := &cobra.Command{
		Use:   "project",
		Short: "Manage AST projects",
	}

	createProjCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new project",
		RunE:  runCreateProjectCommand(projectsWrapper),
	}
	createProjCmd.PersistentFlags().StringP(inputFlag, inputFlagSh, "",
		"The object representing the requested project, in JSON format")
	createProjCmd.PersistentFlags().StringP(inputFileFlag, inputFileFlagSh, "",
		"A file holding the requested project object in JSON format. Takes precedence over --input")

	listProjectsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects in the system",
		RunE:  runListProjectsCommand(projectsWrapper),
	}
	listProjectsCmd.PersistentFlags().Uint64P(limitFlag, limitFlagSh, 0, limitUsage)
	listProjectsCmd.PersistentFlags().Uint64P(offsetFlag, offsetFlagSh, 0, offsetUsage)

	showProjectCmd := &cobra.Command{
		Use:   "show",
		Short: "Show information about a project",
		RunE:  runGetProjectByIDCommand(projectsWrapper),
	}

	deleteProjCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project",
		RunE:  runDeleteProjectCommand(projectsWrapper),
	}

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Get a list of all available tags",
		RunE:  runGetProjectsTagsCommand(projectsWrapper),
	}

	projCmd.AddCommand(createProjCmd, showProjectCmd, listProjectsCmd, deleteProjCmd, tagsCmd)
	return projCmd
}

func runCreateProjectCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var input []byte
		var err error

		var projInputFile string
		var projInput string
		projInput, _ = cmd.Flags().GetString(inputFlag)
		projInputFile, _ = cmd.Flags().GetString(inputFileFlag)

		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFlag, projInput))
		PrintIfVerbose(fmt.Sprintf("%s: %s", inputFileFlag, projInputFile))

		if projInputFile != "" {
			// Reading project from input file
			PrintIfVerbose(fmt.Sprintf("Reading project input from file %s", projInputFile))
			input, err = ioutil.ReadFile(projInputFile)
			if err != nil {
				return errors.Wrapf(err, "%s: Failed to open input file", failedCreatingProj)
			}
		} else if projInput != "" {
			// Reading from standard input
			PrintIfVerbose("Reading project input from console")
			input = bytes.NewBufferString(projInput).Bytes()
		} else {
			// No input was given
			return errors.Errorf("%s: no input was given\n", failedCreatingProj)
		}
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
			err = outputProject(cmd, projResponseModel)
			if err != nil {
				return errors.Wrapf(err, "%s", failedCreatingProj)
			}
		}
		return nil
	}
}

func runListProjectsCommand(projectsWrapper wrappers.ProjectsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allProjectsModel *projectsRESTApi.SlicedProjectsResponseModel
		var errorModel *projectsRESTApi.ErrorModel
		var err error
		limit, offset := getLimitAndOffset(cmd)

		allProjectsModel, errorModel, err = projectsWrapper.Get(limit, offset)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s\n", failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allProjectsModel != nil && allProjectsModel.Projects != nil {
			err = outputProjects(cmd, allProjectsModel)
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
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a project ID", failedGettingProj)
		}
		projectID := args[0]
		projectResponseModel, errorModel, err = projectsWrapper.GetByID(projectID)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingProj)
		}
		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingProj, errorModel.Code, errorModel.Message)
		} else if projectResponseModel != nil {
			err = outputProject(cmd, projectResponseModel)
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
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a project ID", failedDeletingProj)
		}
		projectID := args[0]
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
		var tags *[]string
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

func outputProjects(cmd *cobra.Command, model *projectsRESTApi.SlicedProjectsResponseModel) error {
	if IsJSONFormat() {
		var allProjectsJSON []byte
		allProjectsJSON, err := json.Marshal(model)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to serialize project response ", failedGettingAll)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(allProjectsJSON))
	} else if IsPrettyFormat() {
		for _, project := range model.Projects {
			outputSingleProject(&projectsRESTApi.ProjectResponseModel{
				ID:      project.ID,
				Created: project.Created,
				Updated: project.Updated,
				Tags:    project.Tags,
			})
		}
	}
	return nil
}

func outputProject(cmd *cobra.Command, model *projectsRESTApi.ProjectResponseModel) error {
	if err := ValidateFormat(); err != nil {
		return err
	}

	if IsJSONFormat() {
		responseModelJSON, err := json.Marshal(model)
		if err != nil {
			return errors.Wrapf(err, "Failed to serialize project response")
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(responseModelJSON))
	} else if IsPrettyFormat() {
		outputSingleProject(model)
	}
	return nil
}

func outputSingleProject(model *projectsRESTApi.ProjectResponseModel) {
	fmt.Println("----------------------------")
	fmt.Println("Project ID:", model.ID)
	fmt.Println("Created at:", model.Created)
	fmt.Println("Updated at:", model.Updated)
	fmt.Println("Tags:", model.Tags)
}
