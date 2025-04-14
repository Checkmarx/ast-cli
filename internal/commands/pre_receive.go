package commands

import (
	pre_receive "github.com/Checkmarx/secret-detection/pkg/hooks/pre-receive"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

// PreReceiveCommand creates the pre-receive subcommand
func PreReceiveCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	preReceiveCmd := &cobra.Command{
		Use:   "pre-receive",
		Short: "Manage pre-receive hooks and run secret detection scans",
		Long:  "The pre-receive command enables the ability to manage Git pre-receive hooks for secret detection.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive install
            $ cx hooks pre-receive scan
        `,
		),
	}
	preReceiveCmd.PersistentFlags().Bool("global", false, "Install the hook globally for all repositories")

	preReceiveCmd.AddCommand(installPreReceiveHookCommand(jwtWrapper))
	preReceiveCmd.AddCommand(uninstallPreReceiveHookCommand(jwtWrapper))
	preReceiveCmd.AddCommand(updatePreReceiveHookCommand(jwtWrapper))
	preReceiveCmd.AddCommand(testPreReceiveHookCommand(jwtWrapper))

	return preReceiveCmd
}

func installPreReceiveHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the pre-receive hook",
		Long:  "Install the pre-receive hook for secret detection in your repository.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive install
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return pre_receive.Install(global)
		},
	}

	return cmd
}

func uninstallPreReceiveHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the pre-receive hook",
		Long:  "Uninstall the pre-receive hook for secret detection from your repository.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive uninstall
        `,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return pre_receive.Uninstall(global)
		},
	}

	return cmd
}

func updatePreReceiveHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the pre-receive hook",
		Long:  "Update the pre-receive hook for secret detection to the latest version.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive update
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return pre_receive.Update(global)
		},
	}

	return cmd
}

func testPreReceiveHookCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-hook",
		Short: "Test the pre-receive hook",
		Long:  "Test the pre-receive hook to ensure it is working correctly.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive test-hook
        `,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
