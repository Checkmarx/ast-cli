package commands

import (
	"github.com/MakeNowJust/heredoc"
	errorconsts "github.com/checkmarx/ast-cli/internal/errors"
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
			return errors.Errorf(errorconsts.MissingImportFlags)
		}
		projectName, err := cmd.Flags().GetString(commonParams.ProjectName)
		if err != nil {
			return err
		}
		if projectName == "" {
			return errors.Errorf(errorconsts.ProjectNameIsRequired)
		}

		projectID, err := findProject(nil, projectName, cmd, projectsWrapper, groupsWrapper, accessManagementWrapper)
		if err != nil {
			return err
		}

		_, err = importFile(projectID, importFilePath)
		if err != nil {
			return err
		}

		return nil
	}
}

func importFile(projectID, path string) (string, error) {
	// returns importId as string
	return "", nil
}
