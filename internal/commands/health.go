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

func runHealthCheck(c *wrappers.HealthCheck) {
	status, err := c.Handler()
	if err != nil {
		fmt.Printf("%v error %v\n", c.Name, err)
	} else {
		fmt.Printf("%v status: %v\n", c.Name, status)
	}
}

func runChecksConcurrently(checks []*wrappers.HealthCheck) {
	var wg sync.WaitGroup
	for _, healthChecker := range checks {
		wg.Add(1) //nolint:mnd
		go func(c *wrappers.HealthCheck) {
			defer wg.Done()
			runHealthCheck(c)
		}(healthChecker)
	}

	wg.Wait()
}

func newHealthChecksByRole(h wrappers.HealthCheckWrapper, role string) (checksByRole []*wrappers.HealthCheck) {
	sastRoles := [...]string{commonParams.SastALlInOne, commonParams.SastEngine, commonParams.SastManager}
	sastAndScaRoles := append(sastRoles[:], commonParams.ScaAgent)
	healthChecks := []*wrappers.HealthCheck{
		wrappers.NewHealthCheck("DB", h.RunDBCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Web App", h.RunWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("In-memory DB", h.RunInMemoryDBCheck, sastAndScaRoles),
		wrappers.NewHealthCheck("Object Store", h.RunObjectStoreCheck, sastAndScaRoles),
		wrappers.NewHealthCheck("Message Queue", h.RunMessageQueueCheck, sastAndScaRoles),
	}

	for _, hc := range healthChecks {
		if hc.HasRole(role) {
			checksByRole = append(checksByRole, hc)
		}
	}

	return checksByRole
}

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		hlthChks := newHealthChecksByRole(healthCheckWrapper, viper.GetString(commonParams.AstRoleKey))
		runChecksConcurrently(hlthChks)
	}
}
