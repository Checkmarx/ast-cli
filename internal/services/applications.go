package services

import (
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/pkg/errors"
)

const (
	ApplicationRuleType = "project.name.in"
)

func createApplicationIds(applicationID, existingApplicationIds []string) []string {
	for _, id := range applicationID {
		if !utils.Contains(existingApplicationIds, id) {
			existingApplicationIds = append(existingApplicationIds, id)
		}
	}
	return existingApplicationIds
}

func getApplicationID(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper) ([]string, error) {
	var applicationID []string
	if applicationName != "" {
		application, getAppErr := GetApplication(applicationName, applicationsWrapper)
		if getAppErr != nil {
			return nil, getAppErr
		}
		if application == nil {
			return nil, errors.Errorf(errorConstants.ApplicationDoesntExistOrNoPermission)
		}
		applicationID = []string{application.ID}
	}
	return applicationID, nil
}

func GetApplication(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper) (*wrappers.Application, error) {
	if applicationName != "" {
		params := make(map[string]string)
		params["name"] = applicationName
		resp, err := applicationsWrapper.Get(params)
		if err != nil {
			return nil, err
		}
		if resp.Applications != nil && len(resp.Applications) > 0 {
			application := verifyApplicationNameExactMatch(applicationName, resp)

			return application, nil
		}
	}
	return nil, nil
}

func verifyApplicationNameExactMatch(applicationName string, resp *wrappers.ApplicationsResponseModel) *wrappers.Application {
	var application *wrappers.Application
	for i := range resp.Applications {
		if resp.Applications[i].Name == applicationName {
			application = &resp.Applications[i]
			break
		}
	}
	return application
}

func findApplicationAndUpdate(applicationName string, applicationsWrapper wrappers.ApplicationsWrapper, projectName string) error {
	if applicationName == "" {
		logger.PrintfIfVerbose("No application name provided. Skipping application update")
		return nil
	}
	applicationResp, err := GetApplication(applicationName, applicationsWrapper)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Application:%s", applicationName)
	}
	if applicationResp == nil {
		return errors.Errorf("Application %s not found", applicationName)
	}
	var applicationModel wrappers.ApplicationConfiguration
	var newApplicationRule wrappers.Rule
	var applicationID string

	applicationModel.Name = applicationResp.Name
	applicationModel.Description = applicationResp.Description
	applicationModel.Criticality = applicationResp.Criticality
	applicationModel.Type = applicationResp.Type
	applicationModel.Tags = applicationResp.Tags
	newApplicationRule.Type = ApplicationRuleType
	newApplicationRule.Value = projectName
	applicationModel.Rules = append(applicationModel.Rules, applicationResp.Rules...)
	applicationModel.Rules = append(applicationModel.Rules, newApplicationRule)
	applicationID = applicationResp.ID

	err = updateApplication(&applicationModel, applicationsWrapper, applicationID)
	if err != nil {
		return err
	}
	return nil
}

func updateApplication(applicationModel *wrappers.ApplicationConfiguration, applicationWrapper wrappers.ApplicationsWrapper, applicationID string) error {
	errorModel, err := applicationWrapper.Update(applicationID, applicationModel)
	if errorModel != nil {
		err = errors.Errorf(ErrorCodeFormat, "failed to update application", errorModel.Code, errorModel.Message)
	}
	if errorModel == nil && err == nil {
		logger.PrintIfVerbose("Successfully updated the application")
		return nil
	}
	return err
}
