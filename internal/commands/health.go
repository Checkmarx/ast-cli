package commands

import (
	"fmt"
	"sync"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewHealthCheckCommand(healthCheckWrapper wrappers.HealthCheckWrapper) *cobra.Command {
	return &cobra.Command{
		Use:   "health-check",
		Short: "Run AST health check",
		Run:   runAllHealthChecks(healthCheckWrapper),
	}
}

func runHealthCheck(serviceName string, checker wrappers.HealthChecker) {
	status, err := checker()
	if err != nil {
		fmt.Printf("%v error %v\n", serviceName, err)
	} else {
		fmt.Printf("%v status: %v\n", serviceName, status)
	}
}

func runChecksConcurrently(checks map[string]wrappers.HealthChecker) {
	var wg sync.WaitGroup
	for serviceName, checker := range checks {
		wg.Add(1)
		go func(n string, c wrappers.HealthChecker) {
			defer wg.Done()
			runHealthCheck(n, c)
		}(serviceName, checker)
	}

	wg.Wait()
}

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		runChecksConcurrently(map[string]wrappers.HealthChecker{
			"Web App": healthCheckWrapper.RunWebAppCheck,
			"DB":      healthCheckWrapper.RunDBCheck,
		})
	}
}
