package commands

import (
	"fmt"
	"sync"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"

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

func runHealthCheck(c *wrappers.HealthChecker) {
	status, err := c.Checker()
	if err != nil {
		fmt.Printf("%v error %v\n", c.Name, err)
	} else {
		fmt.Printf("%v status: %v\n", c.Name, status)
	}
}

func runChecksConcurrently(checks []*wrappers.HealthChecker) {
	var wg sync.WaitGroup
	for _, healthChecker := range checks {
		wg.Add(1) //nolint:mnd
		go func(c *wrappers.HealthChecker) {
			defer wg.Done()
			runHealthCheck(c)
		}(healthChecker)
	}

	wg.Wait()
}

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		hlthChks := wrappers.NewHealthChecksByRole(healthCheckWrapper, viper.GetString(commonParams.AstRoleKey))
		runChecksConcurrently(hlthChks)
	}
}
