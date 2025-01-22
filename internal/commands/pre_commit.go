package commands

import (
	precommit "github.com/Checkmarx/secret-detection/pkg/hooks"
	"github.com/spf13/cobra"
)

func PreCommitCommand() *cobra.Command {
	preCommitCmd := &cobra.Command{
		Use:   "pre-commit",
		Short: "Manage pre-commit hooks and run secret detection scans",
	}

	preCommitCmd.AddCommand(installCommand())
	preCommitCmd.AddCommand(uninstallCommand())
	preCommitCmd.AddCommand(updateCommand())
	preCommitCmd.AddCommand(scanCommand())
	preCommitCmd.AddCommand(ignoreCommand())
	preCommitCmd.AddCommand(helpCommand())

	return preCommitCmd
}

func installCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the pre-commit hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return precommit.Install()
		},
	}
}

func uninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the pre-commit hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return precommit.Uninstall()
		},
	}
}

func updateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update the pre-commit hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return precommit.Update()
		},
	}
}

func scanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Run the real-time secret detection scan",
		RunE: func(cmd *cobra.Command, args []string) error {
			return precommit.Scan()
		},
	}
}

func ignoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ignore",
		Short: "Ignore a detected secret",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}

func helpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "Display help for commands",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}
