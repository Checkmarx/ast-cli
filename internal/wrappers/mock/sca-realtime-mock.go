package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ScaRealTimeHTTPMockWrapper struct {
	path string
}

func (s ScaRealTimeHTTPMockWrapper) GetScaVulnerabilitiesPackages(scaRequest []wrappers.ScaDependencyBodyRequest) (
	[]wrappers.ScaVulnerabilitiesResponseModel, *wrappers.WebError, error,
) {
	fmt.Println(s.path)
	fmt.Println(scaRequest)
	return nil, nil, nil
}
