package commands

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	inventoryFlag           = "inventory"
	keyFileFlag             = "key-file"
	installScriptPath       = "./installation-scripts/cluster-install.sh"
	failedInstallingCluster = "Failed installing the cluster"
	failedShowingCluster    = "Failed showing the cluster"
)

func NewClusterCommand() *cobra.Command {
	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage AST cluster",
	}

	installClusterCmd := &cobra.Command{
		Use:   "install",
		Short: "Install a new cluster",
		RunE:  runInstallClusterCommand(),
	}
	installClusterCmd.PersistentFlags().StringP(inventoryFlag, "", "",
		"A path to the inventory file")
	installClusterCmd.PersistentFlags().StringP(keyFileFlag, "", "",
		"A path to the ssh key file")

	showClusterCmd := &cobra.Command{
		Use:   "show",
		Short: "Show AST resources",
		RunE:  runShowClusterCommand(),
	}
	clusterCmd.AddCommand(installClusterCmd, showClusterCmd)
	return clusterCmd
}

func runInstallClusterCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		var keyFile string
		var inventoryFile string

		keyFile, _ = cmd.Flags().GetString(keyFileFlag)
		inventoryFile, _ = cmd.Flags().GetString(inventoryFlag)

		if keyFile == "" && inventoryFile == "" {
			return errors.Errorf("Please provide ssh key and inventory file using the flags: --%s and --%s", keyFileFlag, inventoryFlag)
		}

		if keyFile != "" {
			_, err = ioutil.ReadFile(keyFile)
			if err != nil {
				return errors.Wrapf(err, "Failed to open key file")
			}
		} else {
			return errors.Errorf("Please provide the path to the ssh key using the --%s flag", keyFileFlag)
		}

		if inventoryFile != "" {
			_, err = ioutil.ReadFile(inventoryFile)
			if err != nil {
				return errors.Wrapf(err, "Failed to open inventory file")
			}
		} else {
			return errors.Errorf("Please provide the path to the inventory file using the --%s flag", inventoryFlag)
		}

		cmdSh := &exec.Cmd{
			Path:   installScriptPath,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err = cmdSh.Run()
		if err != nil {
			return errors.Wrapf(err, "%s:Failed to run script", failedInstallingCluster)
		}
		return nil
	}
}

func runShowClusterCommand() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		command := exec.Command("kubectl", "get", "nodes")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			return errors.Wrapf(err, "%s:Failed to run kubectl command", failedShowingCluster)
		}
		return nil
	}
}
