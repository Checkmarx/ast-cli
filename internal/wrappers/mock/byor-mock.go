package mock

import (
	"fmt"

	errors2 "github.com/checkmarx/ast-cli/internal/constants/errors"
)

type ByorMockWrapper struct{}

func (b *ByorMockWrapper) Import(projectID, uploadURL string) (string, error) {
	if projectID == FakeUnauthorized401 {
		return "", fmt.Errorf(errors2.StatusUnauthorized)
	}
	if projectID == FakeForbidden403 {
		return "", fmt.Errorf(errors2.StatusForbidden)
	}
	if projectID == FakeInternalServerError500 {
		return "", fmt.Errorf(errors2.StatusInternalServerError)
	}
	return "", nil
}
