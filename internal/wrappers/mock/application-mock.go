package mock

import (
	"time"

	applicationErrors "github.com/checkmarx/ast-cli/internal/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ApplicationsMockWrapper struct{}

func (a ApplicationsMockWrapper) Get(params map[string]string) (*wrappers.ApplicationsResponseModel, error) {
	if params["name"] == NoPermissionApp {
		return nil, errors.Errorf(applicationErrors.ApplicationDoesntExistOrNoPermission)
	}
	if params["name"] == ApplicationDoesntExist {
		return nil, errors.Errorf(applicationErrors.ApplicationDoesntExistOrNoPermission)
	}
	if params["name"] == FakeHTTPStatusBadRequest {
		return nil, errors.Errorf(applicationErrors.FailedToGetApplication)
	}
	if params["name"] == FakeHTTPStatusInternalServerError {
		return nil, errors.Errorf(applicationErrors.FailedToGetApplication)
	}
	mockApplication := wrappers.Application{
		ID:          "mockID",
		Name:        "MOCK",
		Description: "This is a mock application",
		Criticality: 2,
		ProjectIds:  []string{"ProjectID1", "ProjectID2", "MOCK"},
		CreatedAt:   time.Now(),
	}

	response := &wrappers.ApplicationsResponseModel{
		TotalCount:   1,
		Applications: []wrappers.Application{mockApplication},
	}

	return response, nil
}
