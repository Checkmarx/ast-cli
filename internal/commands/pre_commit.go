package commands

import (
	"fmt"
	precommit "github.com/Checkmarx/secret-detection/pkg/hooks/pre-commit"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"strings"
)

// PreCommitCommand creates the pre-commit subcommand

func PreCommitCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	preCommitCmd := &cobra.Command{
		Use:   "pre-commit",
		Short: "Manage pre-commit hooks and run secret detection scans",
		Long:  "The pre-commit command enables the ability to manage Git pre-commit hooks for secret detection.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-install-git-hook
            $ cx hooks pre-commit secrets-scan
        `,
		),
	}
	preCommitCmd.PersistentFlags().Bool("global", false, "Install the hook globally for all repositories")

	preCommitCmd.AddCommand(secretsInstallGitHookCommand(jwtWrapper))
	preCommitCmd.AddCommand(secretsUninstallGitHookCommand(jwtWrapper))
	preCommitCmd.AddCommand(secretsUpdateGitHookCommand(jwtWrapper))
	preCommitCmd.AddCommand(secretsScanCommand(jwtWrapper))
	preCommitCmd.AddCommand(secretsIgnoreCommand(jwtWrapper))
	preCommitCmd.AddCommand(secretsHelpCommand())

	return preCommitCmd
}

// / validateLicense verifies the user has the required license for secret detection

func secretsInstallGitHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets-install-git-hook",
		Short: "Install the pre-commit hook",
		Long:  "Install the pre-commit hook for secret detection in your repository.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-install-git-hook
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return precommit.Install(global)
		},
	}

	return cmd
}

func secretsUninstallGitHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets-uninstall-git-hook",
		Short: "Uninstall the pre-commit hook",
		Long:  "Uninstall the pre-commit hook for secret detection from your repository.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-uninstall-git-hook
        `,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return precommit.Uninstall(global)
		},
	}

	return cmd
}

func secretsUpdateGitHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets-update-git-hook",
		Short: "Update the pre-commit hook",
		Long:  "Update the pre-commit hook for secret detection to the latest version.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-update-git-hook
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return precommit.Update(global)
		},
	}

	return cmd
}

func secretsScanCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	return &cobra.Command{
		Use:   "secrets-scan",
		Short: "Run the real-time secret detection scan",
		Long:  "Run a real-time scan to detect secrets in your code before committing.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-scan
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return precommit.Scan()
		},
	}
}

func secretsIgnoreCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	var resultIds string
	var all bool

	cmd := &cobra.Command{
		Use:   "secrets-ignore",
		Short: "Ignore one or more detected secrets",
		Long:  "Add detected secrets to the ignore list so they won't be flagged in future scans.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-ignore --resultIds=a1b2c3d4e5f6,f1e2d3c4b5a6
            $ cx hooks pre-commit secrets-ignore --all
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLicense(jwtWrapper); err != nil {
				return err
			}

			if all && len(resultIds) > 0 {
				return fmt.Errorf("--all cannot be used with --resultIds")
			}
			if !all && len(resultIds) == 0 {
				return fmt.Errorf("either --all or --resultIds must be specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return precommit.IgnoreAll()
			}

			// If not ignoring all, process the provided resultIds.
			ids := strings.Split(resultIds, ",")
			for i, id := range ids {
				ids[i] = strings.TrimSpace(id)
			}
			return precommit.Ignore(ids)
		},
	}

	cmd.Flags().StringVar(&resultIds, "resultIds", "", "Comma-separated IDs of results to ignore (e.g., id1,id2,id3)")
	cmd.Flags().BoolVar(&all, "all", false, "Ignore all detected secrets")

	return cmd
}

func secretsHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "secrets-help",
		Short: "Display help for pre-commit commands",
		Long:  "Display detailed information about the pre-commit commands and options.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Parent().Help()
		},
	}
}
