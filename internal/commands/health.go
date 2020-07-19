package commands

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	errorColor          = "\033[1;31m%s\033[0m"
	successColor        = "\033[1;32m%s\033[0m"
	reportBlankPadWidth = 2
	checkMarkCode       = "\u2714\ufe0f"
	crossMarkCode       = "\u274c"
)

func NewHealthCheckCommand(healthCheckWrapper wrappers.HealthCheckWrapper) *cobra.Command {
	return &cobra.Command{
		Use:   "health-check",
		Short: "Run AST health check",
		RunE:  runAllHealthChecks(healthCheckWrapper),
	}
}

func getLongestWidth(checkViews []*healthView) int {
	width := 0
	for _, v := range checkViews {
		if v.Error != nil {
			continue
		}

		for _, s := range v.SubChecks {
			if len(s.Name) > width {
				width = len(s.Name)
			}
		}
	}

	return width
}

func printHealthChecks(checkViews []*healthView) {
	longestWidth := getLongestWidth(checkViews)
	for _, checkView := range checkViews {
		fmt.Println(checkView.Name)
		fmt.Println(strings.Repeat("-", len(checkView.Name)))
		if checkView.Error != nil {
			fmt.Printf(errorColor, checkView.Error)
			fmt.Println()
		} else {
			for _, subCheck := range checkView.SubChecks {
				fmt.Print(subCheck.Name, pad(longestWidth-len(subCheck.Name)+reportBlankPadWidth, ""))
				if subCheck.Success {
					fmt.Printf(successColor, checkMarkCode)
					fmt.Println()
				} else {
					fmt.Printf(errorColor, crossMarkCode)
					fmt.Println()
					for _, e := range subCheck.Errors {
						fmt.Printf(errorColor, e)
						fmt.Println()
					}
				}
			}
		}

		fmt.Print("\n\n")
	}
}

func runChecksConcurrently(checks []*wrappers.HealthCheck) []*healthView {
	var wg sync.WaitGroup
	healthViews := make([]*healthView, len(checks))
	for i, healthChecker := range checks {
		wg.Add(1) //nolint:gomnd
		go func(idx int, c *wrappers.HealthCheck) {
			defer wg.Done()
			status, err := c.Handler()
			h := &healthView{
				c.Name,
				err,
				status,
			}
			healthViews[idx] = h // To avoid race
		}(i, healthChecker)
	}

	wg.Wait()
	return healthViews
}

func newHealthChecksByRole(h wrappers.HealthCheckWrapper, role string) (checksByRole []*wrappers.HealthCheck) {
	sastRoles := [...]string{commonParams.SastALlInOne, commonParams.SastEngine, commonParams.SastManager, "SAST"}
	sastAndScaRoles := append(sastRoles[:], commonParams.ScaAgent, "SCA")
	healthChecks := []*wrappers.HealthCheck{
		wrappers.NewHealthCheck("DB", h.RunDBCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Web App", h.RunWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Identity and Access Management Web App", h.RunKeycloakWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Scan-Flow", h.RunScanFlowCheck, sastRoles[:]),
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
		writeToStandardOutput("Performing health checks...")
		role, _ := cmd.Flags().GetString(roleFlag)
		if role == "" {
			var err error
			role, err = healthCheckWrapper.GetAstRole()
			if err != nil {
				return errors.Wrapf(err, "Failed to get ast role from the ast environment. "+
					"you can set it manually with either the command flags or the cli environment variables")
			}
		}

		hlthChks := newHealthChecksByRole(healthCheckWrapper, role)
		views := runChecksConcurrently(hlthChks)
		printHealthChecks(views)
		return nil
	}
}

type healthView struct {
	Name  string
	Error error
	*wrappers.HealthStatus
}
