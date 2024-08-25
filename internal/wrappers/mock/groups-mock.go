package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type GroupsMockWrapper struct {
}

func (g *GroupsMockWrapper) Get(groupName string) ([]wrappers.Group, error) {
	if groupName == "fake-group-error" {
		return nil, errors.Errorf("fake grroup error")
	}
	if groupName == "existsGroup1" || groupName == "existsGroup2" {
		return []wrappers.Group{{ID: "1", Name: groupName}}, nil
	}
	return []wrappers.Group{{ID: "1", Name: "group"}}, nil
}
