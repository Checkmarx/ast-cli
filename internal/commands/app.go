package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const (
	deployScriptPath = "./installation-scripts/ast-install.sh"
)

func NewAppCommand() *cobra.Command {
	appCmd := &cobra.Command{
		Use:   "app",
		Short: "Manage AST resources",
	}

	deployAppCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy AST into exsiting cluster",
		RunE:  runDeployAppCommand(),
	}

	showAppCmd := &cobra.Command{
		Use:   "show",
		Short: "Show AST resources condition",
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
			fmt.Println(err)
		}
		return nil
	}
}

func runShowAppCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		command := exec.Command("kubectl", "get", "pods")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}
}
