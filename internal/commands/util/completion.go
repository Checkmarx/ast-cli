package util

import (
	"fmt"
	"log"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

const (
	shellFlag = "shell"
	shellSh   = "s"

	failedSetCompletion = "Failed setting completion for shell "
)

func NewCompletionCommand() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: `To load completions:
	
	Bash:
	
		$ source <(cx completion bash)
	
		# To load completions for each session, execute once:
		# Linux:
		$ cx utils completion -s bash > /etc/bash_completion.d/cx
		# macOS:
		$ cx utils completion -s bash > /usr/local/etc/bash_completion.d/cx
	
	Zsh:
	
		# If shell completion is not already enabled in your environment,
		# you will need to enable it.  You can execute the following once:
	
		$ echo "autoload -U compinit; compinit" >> ~/.zshrc
	
		# To load completions for each session, execute once:
		$ cx utils completion -s zsh > "${fpath[1]}/_cx"
	
		# You will need to start a new shell for this setup to take effect.
	
	fish:
	
		$ cx utils completion -s fish | source
	
		# To load completions for each session, execute once:
		$ cx utils completion -s fish > ~/.config/fish/completions/cx.fish
	
	PowerShell:
	
		PS> cx utils completion -s powershell | Out-String | Invoke-Expression
	
		# To load completions for every new session, run:
		PS> cx utils completion -s powershell > cx.ps1
		# and source this file from your PowerShell profile.
	`,
		Args: func(cmd *cobra.Command, args []string) error {
			shellType, _ := cmd.Flags().GetString(shellFlag)

			if shellType == "" || Contains(cmd.ValidArgs, shellType) {
				return nil
			}

			return fmt.Errorf("invalid argument \"%s\" for %s. Allowed values: %s", shellType, shellFlag, cmd.ValidArgs)
		},
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE:      runCompletionCmd(),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.com/resource/documents/en/34965-68653-utils.html#UUID-e086afe1-7bd7-917c-8440-0e965f2e348e
			`),
		},
	}
	completionCmd.PersistentFlags().StringP(shellFlag, shellSh, "", "The type of shell [bash/zsh/fish/powershell]")
	_ = completionCmd.MarkPersistentFlagRequired(shellFlag)

	return completionCmd
}

func runCompletionCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		shellType, _ := cmd.Flags().GetString(shellFlag)

		if shellType == "bash" {
			err = cmd.Root().GenBashCompletion(os.Stdout)
		} else if shellType == "zsh" {
			err = cmd.Root().GenZshCompletion(os.Stdout)
		} else if shellType == "fish" {
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		} else if shellType == "powershell" {
			err = cmd.Root().GenPowerShellCompletion(os.Stdout)
		}

		if err != nil {
			log.Fatal(failedSetCompletion, shellType)
		}

		return nil
	}
}
