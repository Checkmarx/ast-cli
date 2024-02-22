package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type AccessManagementMockWrapper struct{}

func (a AccessManagementMockWrapper) CreateGroupsAssignment(projectID, projectName string, groups []*wrappers.Group) error {
	fmt.Println("Called CreateGroupsAssignment in AccessManagementMockWrapper")
	return nil
}

func (a AccessManagementMockWrapper) GetGroups(projectID string) ([]*wrappers.Group, error) {
	fmt.Println("Called GetGroups in AccessManagementMockWrapper")
	return nil, nil
}
