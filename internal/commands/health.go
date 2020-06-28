package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	scanCompletedTimeoutSecs = 60
	scanProjectID            = "health"
)

func NewHealthCheckCommand(healthCheckWrapper wrappers.HealthCheckWrapper,
	scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper, scanHealthCheckSourcePath string) *cobra.Command {
	return &cobra.Command{
		Use:   "health-check",
		Short: "Run AST health check",
		RunE:  runAllHealthChecks(healthCheckWrapper, scansWrapper, uploadsWrapper, projectsWrapper, scanHealthCheckSourcePath),
	}
}

func runHealthCheck(c *wrappers.HealthCheck) *healthView {
	status, err := c.Handler()
	v := &healthView{Name: c.Name}
	if err != nil {
		v.Status = "Error"
		v.Errors = []string{err.Error()}
	} else if status.Success {
		v.Status = "Success"
	} else {
		v.Status = "Failure"
		v.Errors = status.Errors
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

func checkScanCompleted(scansWrapper wrappers.ScansWrapper, scanID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(scanCompletedTimeoutSecs)*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout exceeded")
		default:
			scan, errorResponse, err := scansWrapper.GetByID(scanID)
			if err != nil {
				return errors.Wrapf(err, "Failed to get scan")
			}

			if errorResponse != nil {
				return errors.Errorf("get scan response CODE: %d %s", errorResponse.Code, errorResponse.Message)
			}

			switch scan.Status {
			case scansRESTApi.ScanFailed:
				return errors.New("scan failed")
			case scansRESTApi.ScanCanceled:
				return errors.New("scan canceled")
			case scansRESTApi.ScanCompleted:
				return nil
			default:
				time.Sleep(time.Second)
			}
		}
	}
}

func scanHealthCheck(scansWrapper wrappers.ScansWrapper,
	uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper, sourcePath string) func() (*wrappers.HealthStatus, error) {
	return func() (status *wrappers.HealthStatus, err error) {
		status = &wrappers.HealthStatus{}
		preSignedURL, err := uploadsWrapper.UploadFile(sourcePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to upload source file")
		}

		scanInput := scansRESTApi.Scan{
			Config: []scansRESTApi.Config{
				{
					Type: "sast",
					Value: map[string]string{
						"presetName": "WordPress",
					},
				},
			},
			Project: scansRESTApi.Project{
				ID:   scanProjectID,
				Type: scansRESTApi.UploadProject,
			},
		}

		projHandlerJSON, err := json.Marshal(scansRESTApi.UploadProjectHandler{
			URL: *preSignedURL,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to serialize scan project handler")
		}

		scanInput.Project.Handler = projHandlerJSON
		scanResponse, errorResponse, err := scansWrapper.Create(&scanInput)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed crating scan")
		}

		if errorResponse != nil {
			status.Errors = append(status.Errors,
				errors.Errorf("Scan response CODE: %d %s", errorResponse.Code, errorResponse.Message).Error(),
			)
			return status, nil
		}

		// TODO Delete uploaded file as-well
		// Delete scan and project
		defer func() {
			errorResponse, errDelete := scansWrapper.Delete(scanResponse.ID)
			if status != nil {
				if errDelete != nil {
					status = nil
					err = errors.Wrapf(err, "Failed to delete scan")
				} else if errorResponse != nil {
					status.Success = false
					status.Errors = append(status.Errors,
						errors.Errorf("Scan delete response CODE: %d %s",
							errorResponse.Code, errorResponse.Message).Error(),
					)
				}
			}

			projErrorResponse, errDelete := projectsWrapper.Delete(scanProjectID)
			if status != nil {
				if errDelete != nil {
					status = nil
					err = errors.Wrapf(err, "Failed to delete scan project")
				} else if projErrorResponse != nil {
					status.Success = false
					status.Errors = append(status.Errors,
						errors.Errorf("Scan project delete response CODE: %d, %s",
							errorResponse.Code, errorResponse.Message).Error())
				}
			}
		}()

		err = checkScanCompleted(scansWrapper, scanResponse.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "Scan was not completed")
		}

		status.Success = true
		return status, nil
	}
}

func newHealthChecksByRole(h wrappers.HealthCheckWrapper, scansWrapper wrappers.ScansWrapper,
	uploadesWrapper wrappers.UploadsWrapper, projectsWrapper wrappers.ProjectsWrapper, scanHealthCheckSourcePath,
	role string) (checksByRole []*wrappers.HealthCheck) {
	sastRoles := [...]string{commonParams.SastALlInOne, commonParams.SastEngine, commonParams.SastManager, "SAST"}
	sastAndScaRoles := append(sastRoles[:], commonParams.ScaAgent, "SCA")
	healthChecks := []*wrappers.HealthCheck{
		wrappers.NewHealthCheck("DB", h.RunDBCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Web App", h.RunWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Keycloak Web App", h.RunKeycloakWebAppCheck, sastRoles[:]),
		wrappers.NewHealthCheck("Scan Flow",
			scanHealthCheck(scansWrapper, uploadesWrapper, projectsWrapper, scanHealthCheckSourcePath), sastRoles[:]),
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

func runAllHealthChecks(healthCheckWrapper wrappers.HealthCheckWrapper,
	scansWrapper wrappers.ScansWrapper, uploadsWrapper wrappers.UploadsWrapper,
	projectsWrapper wrappers.ProjectsWrapper,
	scanHealthCheckSourcePath string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		writeToStandardOutput("Performing health checks...")
		role := viper.GetString(commonParams.AstRoleKey)
		if role == "" {
			var err error
			role, err = healthCheckWrapper.GetAstRole()
			if err != nil {
				return errors.Wrapf(err, "Failed to get ast role from the ast environment. "+
					"you can set it manually with either the command flags or the cli environment variables")
			}
		}

		hlthChks := newHealthChecksByRole(healthCheckWrapper, scansWrapper, uploadsWrapper, projectsWrapper,
			scanHealthCheckSourcePath, role)
		views := runChecksConcurrently(hlthChks)
		fmt.Println("Finished checks", views)
		err := Print(cmd.OutOrStdout(), views)
		return err
	}
}

type healthView struct {
	Name   string
	Status string
	Errors []string
}
