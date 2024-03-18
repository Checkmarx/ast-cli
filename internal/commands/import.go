package commands

import (
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/constants"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper) *cobra.Command {
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
		RunE: runImportCommand(projectsWrapper, uploadsWrapper, groupsWrapper, accessManagementWrapper),
	}

	cmd.PersistentFlags().String(commonParams.ImportFilePath, "", "Path to the import file (sarif file or zip archive containing sarif files)")
	cmd.PersistentFlags().String(commonParams.ProjectName, "", "The project under which the file will be imported.")

	return cmd
}

func runImportCommand(
	projectsWrapper wrappers.ProjectsWrapper,
	_ wrappers.UploadsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	accessManagementWrapper wrappers.AccessManagementWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		importFilePath, err := cmd.Flags().GetString(commonParams.ImportFilePath)
		if err != nil {
			return err
		}
		if importFilePath == "" {
			return errors.Errorf(constants.ImportFilePathIsRequired)
		}

		if err = validateFileExtension(importFilePath); err != nil {
			return err
		}

		projectName, err := getProjectName(cmd)
		if err != nil {
			return err
		}

		projectID, err := findProject(nil, projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper)
		if err != nil {
			return err
		}

		if _, err = importFile(projectID, importFilePath); err != nil {
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
		return "", errors.Errorf(constants.ProjectNameIsRequired)
	}
	return projectName, nil
}

func validateFileExtension(importFilePath string) error {
	extension := filepath.Ext(importFilePath)
	extension = strings.ToLower(extension)
	if extension != constants.SarifExtension && extension != constants.ZipExtension {
		return errors.Errorf(constants.SarifInvalidFileExtension)
	}
	return nil
}

func importFile(projectID, path string) (string, error) {
	// returns importId as string
	return "", nil
}
