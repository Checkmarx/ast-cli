package mock

import (
	"time"

	errorconsts "github.com/checkmarx/ast-cli/internal/constants"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ApplicationsMockWrapper struct{}

func (a ApplicationsMockWrapper) Get(params map[string]string) (*wrappers.ApplicationsResponseModel, error) {
	if params["name"] == NoPermissionApp {
		return nil, errors.Errorf(errorconsts.ApplicationDoesntExistOrNoPermission)
	}
	if params["name"] == ApplicationDoesntExist {
		return nil, errors.Errorf(errorconsts.ApplicationDoesntExistOrNoPermission)
	}
	if params["name"] == FakeBadRequest400 {
		return nil, errors.Errorf(errorconsts.FailedToGetApplication)
	}
	if params["name"] == FakeInternalServerError500 {
		return nil, errors.Errorf(errorconsts.FailedToGetApplication)
	}
	mockApplication := wrappers.Application{
		ID:          "mockID",
		Name:        "MOCK",
		Description: "This is a mock application",
		Criticality: 2,
		ProjectIds:  []string{"ProjectID1", "ProjectID2"},
		CreatedAt:   time.Now(),
	}

	response := &wrappers.ApplicationsResponseModel{
		TotalCount:   1,
		Applications: []wrappers.Application{mockApplication},
	}

	return response, nil
}
