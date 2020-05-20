package commands

import (
	"fmt"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewHealthCommand(wrapper wrappers.HealthcheckWrapper) *cobra.Command {
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Show health information for AST",
		RunE:  runHealthChecks(wrapper),
	}
	return healthCmd
}

func runHealthChecks(wrapper wrappers.HealthcheckWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		healthChecks := []*HealthcheckPhase{
			{
				Name:    "AST Web Application Health Check",
				Checker: wrapper.CheckWebAppIsUp,
			},
			{
				Name:    "AST Database Health Check",
				Checker: wrapper.CheckDatabaseHealth,
			},
		}

		numOfPhases := len(healthChecks)
		for i := 0; i < numOfPhases; i++ {
			phase := healthChecks[i]
			fmt.Println(fmt.Sprintf("Running %d/%d:", i+1, numOfPhases), phase.Name) // nolint
			err := phase.Checker()
			if err != nil {
				fmt.Printf("Failed running %s: %s\n", phase.Name, err.Error())
			} else {
				fmt.Println("Success!")
			}
		}
		return nil
	}
}

type HealthcheckPhase struct {
	Name    string
	Checker func() error
}
