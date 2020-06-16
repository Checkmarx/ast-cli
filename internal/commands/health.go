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

func newHealthChecksByRole(h wrappers.HealthCheckWrapper, role string) (checksByRole []*wrappers.HealthChecker) {
	sastRoles := [...]string{commonParams.SastALlInOne, commonParams.SastEngine, commonParams.SastManager}
	sastAndScaRoles := append(sastRoles[:], commonParams.ScaAgent)
	healthChecks := []*wrappers.HealthChecker{
		(&wrappers.HealthChecker{Name: "DB", Checker: h.RunDBCheck}).AllowRoles(sastRoles[:]...),
		(&wrappers.HealthChecker{Name: "Web App", Checker: h.RunWebAppCheck}).AllowRoles(sastRoles[:]...),
		// TODO replace Redis with TBD
		(&wrappers.HealthChecker{Name: "Redis", Checker: h.RunRedisCheck}).AllowRoles(sastRoles[:]...),
		// TODO replace MinIO with Object Store
		(&wrappers.HealthChecker{Name: "Minio", Checker: h.RunMinioCheck}).AllowRoles(sastAndScaRoles...),
		// TODO replace Nats with Message Queue
		(&wrappers.HealthChecker{Name: "Nats", Checker: h.RunNatsCheck}).AllowRoles(sastAndScaRoles...),
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
