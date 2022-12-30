package scarealtime

import (
	"github.com/spf13/cobra"

	"fmt"
	"os/exec"
)

// RunScaRealtime Main method responsible to run sca realtime feature
func RunScaRealtime() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fmt.Println("Handling SCA Resolver...")
		scaResolverExecutableFile, err := getScaResolver()
		if err != nil {
			return err
		}

		err = executeSCAResolver(scaResolverExecutableFile)
		if err != nil {
			return err
		}

		return nil
	}
}

// executeSCAResolver Executes sca resolver for a specific path
func executeSCAResolver(executable string) error {
	args := []string{
		"offline",
		"-s",
		temporaryProjectPathToScan,
		"-n",
		"dev_sca_realtime_project",
		"-r",
		scaResolverWorkingDir,
	}
	fmt.Println(fmt.Printf("Running SCA resolver with args: %v", args))

	_, err := exec.Command(executable, args...).Output()
	if err != nil {
		return err
	}
	fmt.Println("SCA Resolver finished successfully!")

	return nil
}
