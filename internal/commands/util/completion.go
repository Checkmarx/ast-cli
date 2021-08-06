package util

import (
	"github.com/MakeNowJust/heredoc"
	"log"
	"os"

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
		$ cx completion bash > /etc/bash_completion.d/cx
		# macOS:
		$ cx completion bash > /usr/local/etc/bash_completion.d/cx
	
	Zsh:
	
		# If shell completion is not already enabled in your environment,
		# you will need to enable it.  You can execute the following once:
	
		$ echo "autoload -U compinit; compinit" >> ~/.zshrc
	
		# To load completions for each session, execute once:
		$ cx completion zsh > "${fpath[1]}/_cx"
	
		# You will need to start a new shell for this setup to take effect.
	
	fish:
	
		$ cx completion fish | source
	
		# To load completions for each session, execute once:
		$ cx completion fish > ~/.config/fish/completions/cx.fish
	
	PowerShell:
	
		PS> cx completion powershell | Out-String | Invoke-Expression
	
		# To load completions for every new session, run:
		PS> cx completion powershell > cx.ps1
		# and source this file from your PowerShell profile.
	`,
		RunE: runCompletionCmd(),
		Annotations: map[string]string{
			"utils:env": heredoc.Doc(`
				See 'cx utils env' for the list of supported environment variables	
			`),
			"command:doc": heredoc.Doc(`
				https://checkmarx.atlassian.net/wiki/x/gwQRtw
			`),
		},
	}
	completionCmd.PersistentFlags().StringP(shellFlag, shellSh, "", "The type of shell [bash/zsh/fish/powershell]")
	completionCmd.MarkPersistentFlagRequired(shellFlag)

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
