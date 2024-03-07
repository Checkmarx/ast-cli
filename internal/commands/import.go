package commands

import (
	"github.com/MakeNowJust/heredoc"
	clierrors "github.com/checkmarx/ast-cli/internal/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewImportCommand(uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
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
		RunE: runImportCommand(uploadsWrapper),
	}

	cmd.PersistentFlags().String(commonParams.ImportFileType, "", "The type of the imported file (SARIF or ZIP containing SARIF files)")
	cmd.PersistentFlags().String(commonParams.ImportFilePath, "", "The local path of the imported file")

	return cmd
}

func runImportCommand(wrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		importFileType, err := cmd.Flags().GetString(commonParams.ImportFileType)
		if err != nil {
			return err
		}
		importFilePath, err := cmd.Flags().GetString(commonParams.ImportFilePath)
		if err != nil {
			return err
		}

		if importFileType == "" || importFilePath == "" {
			return errors.Errorf(clierrors.MissingImportFlags)
		}
		_, err = importFile(importFileType, importFilePath)
		if err != nil {
			return err
		}

		return nil
	}
}

func importFile(fileType string, path string) (string, error) {
	// returns importId as string
	return "", nil
}
