package commands

import (
	"log"
	"os"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewUtilsCommand(healthCheckWrapper wrappers.HealthCheckWrapper) *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "utils",
		Short: "AST Utility functions",
	}
	healthCheckCmd := NewHealthCheckCommand(healthCheckWrapper)
	envCheckCmd := NewEnvCheckCommand()
	var completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
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
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				err = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	scanCmd.AddCommand(healthCheckCmd, completionCmd, envCheckCmd)

	return scanCmd
}
