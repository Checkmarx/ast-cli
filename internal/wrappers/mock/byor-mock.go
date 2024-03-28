package mock

import (
	"fmt"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
)

type ByorMockWrapper struct{}

func (b *ByorMockWrapper) Import(projectID, uploadURL string) (string, error) {
	if projectID == FakeUnauthorized401 {
		return "", fmt.Errorf(errorConstants.StatusUnauthorized)
	}
	if projectID == FakeForbidden403 {
		return "", fmt.Errorf(errorConstants.StatusForbidden)
	}
	if projectID == FakeInternalServerError500 {
		return "", fmt.Errorf(errorConstants.StatusInternalServerError)
	}
	return "", nil
}
