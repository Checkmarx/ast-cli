package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type CustomStatesMockWrapper struct{}

func (m *CustomStatesMockWrapper) GetAllCustomStates(includeDeleted bool) ([]wrappers.CustomState, error) {
	if includeDeleted {
		return []wrappers.CustomState{
			{ID: 1, Name: "demo1", Type: "Custom"},
			{ID: 2, Name: "demo2", Type: "System"},
			{ID: 3, Name: "demo3", Type: "Custom"},
		}, nil
	}
	return []wrappers.CustomState{
		{ID: 2, Name: "demo2", Type: "System"},
		{ID: 3, Name: "demo3", Type: "Custom"},
	}, nil
}