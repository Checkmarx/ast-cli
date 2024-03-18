package mock

import (
	"fmt"
)

type ByorMockWrapper struct{}

func (b *ByorMockWrapper) Import(projectID, uploadURL string) (string, error) {
	fmt.Println("Called Import in ByorMockWrapper")
	return "", nil
}
