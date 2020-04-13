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

	// installClusterCmd := &cobra.Command{
	// 	Use:   "install",
	// 	Short: "Install a new cluster",
	// 	RunE:  runInstallClusterCommand(projectsWrapper),
	// }

	// lsClusterCmd := &cobra.Command{
	// 	Use:   "ls",
	// 	Short: "Returns cluster information",
	// 	RunE:  runLsClusterCommand(projectsWrapper),
	// }

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
