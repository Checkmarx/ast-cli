package services

import (
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/pkg/errors"
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
