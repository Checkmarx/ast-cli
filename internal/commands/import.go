package commands

import (
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/constants"
	"github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	byorWrapper wrappers.ByorWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import SAST scan results",
		Long:  "The import command enables you to import SAST scan results from an external source into Checkmarx One. The results must be submitted in sarif format.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68625-checkmarx-one-cli-commands.html
			`,
			),
		},
		RunE: runImportCommand(projectsWrapper, uploadsWrapper, groupsWrapper, accessManagementWrapper, byorWrapper),
	}

	cmd.PersistentFlags().String(commonParams.ImportFilePath, "", "Path to the import file (sarif file or zip archive containing sarif files)")
	cmd.PersistentFlags().String(commonParams.ProjectName, "", "The project under which the file will be imported.")

	return cmd
}

func runImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper,
	byorWrapper wrappers.ByorWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		importFilePath, err := cmd.Flags().GetString(commonParams.ImportFilePath)
		if err != nil {
			return err
		}
		if importFilePath == "" {
			return errors.Errorf(errorconstants.ImportFilePathIsRequired)
		}

		if validationError := validateFileExtension(importFilePath); validationError != nil {
			return validationError
		}

		projectName, err := getProjectName(cmd)
		if err != nil {
			return err
		}

		projectID, err := findProject(nil, projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper)
		if err != nil {
			return err
		}

		_, err = importFile(projectID, importFilePath, uploadsWrapper, byorWrapper)
		if err != nil {
			return err
		}

		return nil
	}
}

func getProjectName(cmd *cobra.Command) (string, error) {
	projectName, err := cmd.Flags().GetString(commonParams.ProjectName)
	if err != nil {
		return "", err
	}
	if projectName == "" {
		return "", errors.Errorf(errorconstants.ProjectNameIsRequired)
	}
	return projectName, nil
}

func validateFileExtension(importFilePath string) error {
	extension := filepath.Ext(importFilePath)
	extension = strings.ToLower(extension)
	if extension != constants.SarifExtension && extension != constants.ZipExtension {
		return errors.Errorf(errorconstants.SarifInvalidFileExtension)
	}
	return nil
}

func importFile(projectID string, path string,
	uploadsWrapper wrappers.UploadsWrapper, byorWrapper wrappers.ByorWrapper) (string, error) {
	uploadURL, err := uploadsWrapper.UploadFile(path)
	if err != nil {
		return "", err
	}
	importID, err := byorWrapper.Import(projectID, *uploadURL)
	if err != nil {
		return "", err
	}
	return importID, nil
}
