package commands

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewHealthCheckCommand(healthCheckWrapper wrappers.HealthCheckWrapper) *cobra.Command {
	return &cobra.Command{
		Use:   "health-check",
		Short: "Run AST health check",
		RunE:  runAllHealthChecks(healthCheckWrapper),
	}
}

func runWebAppHealthCheck(healthCheckWrapper wrappers.HealthCheckWrapper) error {
	status, err := healthCheckWrapper.RunWebAppCheck()
	if err != nil {
		return errors.Wrapf(err, "Web app error")
	}

	fmt.Printf("Web App status: %v", status)
	return nil
}

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runWebAppHealthCheck(healthCheckWrapper)
	}
}
