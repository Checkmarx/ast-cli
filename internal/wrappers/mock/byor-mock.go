package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/constants"
)

type ByorMockWrapper struct{}

func (b *ByorMockWrapper) Import(projectID, uploadURL string) (string, error) {
	if projectID == FakeUnauthorized401 {
		return "", fmt.Errorf(constants.StatusUnauthorized)
	}
	if projectID == FakeForbidden403 {
		return "", fmt.Errorf(constants.StatusForbidden)
	}
	if projectID == FakeInternalServerError500 {
		return "", fmt.Errorf(constants.StatusInternalServerError)
	}
	return "", nil
}
