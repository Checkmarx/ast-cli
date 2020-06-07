package commands

import (
	"fmt"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewHealthCommand(healthWrapper wrappers.HealthWrapper) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Execute health check",
		RunE:  runHealthChecks(healthWrapper),
	}
}

func runWebAppHealthCheck(healthWrapper wrappers.HealthWrapper) error {
	err := healthWrapper.RunWebAppCheck()
	if err == nil {
		fmt.Printf("Web App status: OK\n")
	}

	return err

}

func runHealthChecks(healthWrapper wrappers.HealthWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runWebAppHealthCheck(healthWrapper)
	}
}
