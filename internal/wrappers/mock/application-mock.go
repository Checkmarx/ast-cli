package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"time"
)

type ApplicationsMockWrapper struct{}

func (a ApplicationsMockWrapper) Get(params map[string]string) (*wrappers.ApplicationsResponseModel, *wrappers.ErrorModel, error) {
	if params["application-name"] == "NoPermissionApp" {
		return nil, nil, errors.Errorf("project doesn’t exists, no permission to create project")
	}
	mockApplication := wrappers.Application{
		Id:          "mockID",
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

	return response, nil, nil
}
