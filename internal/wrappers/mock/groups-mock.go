package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type GroupsMockWrapper struct {
}

func (g *GroupsMockWrapper) Get(_ string) ([]wrappers.Group, error) {
	return []wrappers.Group{{ID: "1", Name: "group"}}, nil
}
