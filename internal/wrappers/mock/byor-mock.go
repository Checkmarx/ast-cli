package mock

import (
	"fmt"
)

type ByorMockWrapper struct{}

func (b *ByorMockWrapper) Import(projectID, uploadURL string) (string, error) {
	if projectID == FakeUnauthorized401 {
		return "", fmt.Errorf(errors.StatusUnauthorized)
	}
	if projectID == FakeForbidden403 {
		return "", fmt.Errorf(errors.StatusForbidden)
	}
	if projectID == FakeInternalServerError500 {
		return "", fmt.Errorf(errors.StatusInternalServerError)
	}
	return "", nil
}
