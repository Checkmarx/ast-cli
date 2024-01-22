package util

import (
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewMaskSecretsCommand(chatWrapper wrappers.ChatWrapper) *cobra.Command {
	maskSecretsCmd := &cobra.Command{
		Use:   "mask",
		Short: "Mask secrets in a file",
		Long: `To Mask secrets in a file
	`,
		RunE: runMaskSecretCmd(chatWrapper),
		Example: heredoc.Doc(
			`
			$ cx utils mask --results-file <resultsFile> 
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`https://checkmarx.com/resource/documents/en/34965-68653-utils.html
			`,
			),
		},
	}
	maskSecretsCmd.Flags().String(params.ChatKicsResultFile, "", "IaC result code file")
	_ = maskSecretsCmd.MarkFlagRequired(params.ChatKicsResultFile)

	return maskSecretsCmd
}

func runMaskSecretCmd(chatWrapper wrappers.ChatWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error
		unmaskedFile, _ := cmd.Flags().GetString(params.ChatKicsResultFile)
		unmaskedContent, err := os.ReadFile(unmaskedFile)
		if err != nil {
			return errors.Errorf("Error opening file : %s", err.Error())
		}
		statefulWrapper := wrapper.NewStatefulWrapper(connector.NewFileSystemConnector(""), "", "", 0, 0)
		maskedEntry, err := chatWrapper.MaskSecrets(statefulWrapper, string(unmaskedContent))
		if err != nil {
			return err
		}
		err = printer.Print(cmd.OutOrStdout(), maskedEntry, printer.FormatJSON)
		if err != nil {
			return err
		}
		return nil
	}
}
