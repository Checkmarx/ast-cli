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
		RunE:  runAllHealthChecks(healthCheckWrapper),
	}
}

func runHealthCheck(c *wrappers.HealthCheck) *healthView {
	status, err := c.Handler()
	v := &healthView{Name: c.Name}
	if err != nil {
		v.Status = fmt.Sprintf("Error %s", err)
	} else {
		v.Status = status.String()
	}

	return v
}

func runChecksConcurrently(checks []*wrappers.HealthCheck) []*healthView {
	var wg sync.WaitGroup
	healthViews := make([]*healthView, len(checks))
	for i, healthChecker := range checks {
		wg.Add(1) //nolint:gomnd
		go func(idx int, c *wrappers.HealthCheck) {
			defer wg.Done()
			h := runHealthCheck(c)
			healthViews[idx] = h // To avoid race
		}(i, healthChecker)
	}

	wg.Wait()
	return healthViews
}

func newHealthChecksByRole(h wrappers.HealthCheckWrapper, role string) (checksByRole []*wrappers.HealthCheck) {
	sastRoles := [...]string{commonParams.SastALlInOne, commonParams.SastEngine, commonParams.SastManager}
	sastAndScaRoles := append(sastRoles[:], commonParams.ScaAgent)
	healthChecks := []*wrappers.HealthCheck{
		wrappers.NewHealthCheck("DB", h.RunDBCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Web App", h.RunWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Keycloak Web App", h.RunKeycloakWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("In-memory DB", h.RunInMemoryDBCheck, sastAndScaRoles),
		wrappers.NewHealthCheck("Object Store", h.RunObjectStoreCheck, sastAndScaRoles),
		wrappers.NewHealthCheck("Message Queue", h.RunMessageQueueCheck, sastAndScaRoles),
		wrappers.NewHealthCheck("Logging", h.RunLoggingCheck, sastAndScaRoles),
	}

	for _, hc := range healthChecks {
		if hc.HasRole(role) {
			checksByRole = append(checksByRole, hc)
		}
	}

	return checksByRole
}

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		role := viper.GetString(commonParams.AstRoleKey)
		if role == "" {
			var err error
			role, err = healthCheckWrapper.GetAstRole()
			if err != nil {
				return err
			}
		}

		hlthChks := newHealthChecksByRole(healthCheckWrapper, role)
		views := runChecksConcurrently(hlthChks)
		err := Print(cmd.OutOrStdout(), views)
		return err
	}
}

type healthView struct {
	Name   string
	Status string
}
