package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewClusterCommand() *cobra.Command {
	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage AST cluster",
	}

	deployClusterCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy AST resources",
		RunE:  runDeployClusterCommand(),
	}
	clusterCmd.AddCommand(deployClusterCmd)
	return clusterCmd
}

func runDeployClusterCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Println("deploy cluster")
		return nil
	}
}
