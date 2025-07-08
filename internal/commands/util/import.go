package util

import (
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/constants"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	byorWrapper wrappers.ByorWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import SAST scan results",
		Long:  "The import command enables you to import SAST scan results from an external source into Checkmarx One. The results must be submitted in sarif format",
		Example: heredoc.Doc(
			`
			$ cx utils import --project-name <project name>  --import-file-path <file path>
		`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68625-checkmarx-one-cli-commands.html
			`,
			),
		},
		RunE: runImportCommand(projectsWrapper, uploadsWrapper, groupsWrapper, applicationsWrapper, byorWrapper, featureFlagsWrapper),
	}

	cmd.PersistentFlags().String(commonParams.ImportFilePath, "", "Path to the import file (sarif file or zip archive containing sarif files)")
	cmd.PersistentFlags().String(commonParams.ProjectName, "", "The project under which the file will be imported")

	return cmd
}

func runImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	applicationsWrapper wrappers.ApplicationsWrapper,
	byorWrapper wrappers.ByorWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		importFilePath, err := validateFilePath(cmd)
		if err != nil {
			return err
		}

		projectName, _ := cmd.Flags().GetString(commonParams.ProjectName)
		if projectName == "" {
			return errors.Errorf(errorConstants.ProjectNameIsRequired)
		}

		projectID, err := services.FindProject(projectName, cmd, projectsWrapper, groupsWrapper, applicationsWrapper)
		if err != nil {
			return err
		}

		err = importFile(projectID, importFilePath, uploadsWrapper, byorWrapper, featureFlagsWrapper)
		if err != nil {
			return err
		}

		return nil
	}
}

func validateFilePath(cmd *cobra.Command) (string, error) {
	importFilePath, _ := cmd.Flags().GetString(commonParams.ImportFilePath)
	if importFilePath == "" {
		return "", errors.Errorf(errorConstants.ImportFilePathIsRequired)
	}

	if validationError := validateFileExtension(importFilePath); validationError != nil {
		return "", validationError
	}
	return importFilePath, nil
}

func validateFileExtension(importFilePath string) error {
	extension := filepath.Ext(importFilePath)
	extension = strings.ToLower(extension)
	if extension != constants.SarifExtension && extension != constants.ZipExtension {
		return errors.Errorf(errorConstants.SarifInvalidFileExtension)
	}
	return nil
}

func importFile(projectID string, path string,
	uploadsWrapper wrappers.UploadsWrapper, byorWrapper wrappers.ByorWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	uploadURL, err := uploadsWrapper.UploadFile(path, featureFlagsWrapper)
	if err != nil {
		return err
	}
	_, err = byorWrapper.Import(projectID, *uploadURL)
	if err != nil {
		return err
	}
	return nil
}
