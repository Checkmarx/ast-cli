package commands

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deployScriptPath   = "./installation-scripts/ast-install.sh"
	failedDeployingApp = "Failed deploying app"
	failedShowingApp   = "Failed showing app"
)

func NewAppCommand() *cobra.Command {
	appCmd := &cobra.Command{
		Use:   "app",
		Short: "Manage resources",
	}

	deployAppCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy into existing cluster",
		RunE:  runDeployAppCommand(),
	}

	showAppCmd := &cobra.Command{
		Use:   "show",
		Short: "Show resources state",
		RunE:  runShowAppCommand(),
	}
	appCmd.AddCommand(deployAppCmd, showAppCmd)
	return appCmd
}

func runDeployAppCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		cmdSh := &exec.Cmd{
			Path:   deployScriptPath,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err = cmdSh.Run()
		if err != nil {
			return errors.Wrapf(err, "%s:Failed to run script", failedDeployingApp)
		}
		return nil
	}
}

func runShowAppCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_, _, err := runBashCommand("kubectl", []string{}, "get", "pods")
		if err != nil {
			return errors.Wrapf(err, "%s:Failed to run kubectl command", failedShowingApp)
		}
		return nil
	}
}
