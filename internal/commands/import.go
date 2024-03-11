package commands

import (
	"github.com/MakeNowJust/heredoc"
	clierrors "github.com/checkmarx/ast-cli/internal/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewImportCommand(projectsWrapper wrappers.ProjectsWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import scan results",
		Long:  "Import a SARIF file or a ZIP file containing SARIF file/s.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68625-checkmarx-one-cli-commands.html
			`,
			),
		},
		RunE: runImportCommand(projectsWrapper, uploadsWrapper),
	}

	cmd.PersistentFlags().String(commonParams.ImportFilePath, "", "The local path of the imported file")
	cmd.PersistentFlags().String(commonParams.ProjectName, "", "The project under which the file will be imported.")

	return cmd
}

func runImportCommand(projectsWrapper wrappers.ProjectsWrapper, _ wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		importFilePath, err := cmd.Flags().GetString(commonParams.ImportFilePath)
		if err != nil {
			return err
		}
		if importFilePath == "" {
			return errors.Errorf(clierrors.MissingImportFlags)
		}
		projectName, err := cmd.Flags().GetString(commonParams.ProjectName)
		if err != nil {
			return err
		}
		if projectName == "" {
			return errors.Errorf(clierrors.ProjectNameIsRequired)
		}

		project, _, err := projectsWrapper.GetByName(projectName)

		_, err = importFile(project.ID, importFilePath)
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
