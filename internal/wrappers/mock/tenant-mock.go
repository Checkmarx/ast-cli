package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type TenantConfigurationMockWrapper struct {
}

func (t TenantConfigurationMockWrapper) GetTenantConfiguration() (
	*[]*wrappers.TenantConfigurationResponse,
	*wrappers.WebError,
	error,
) {
	return nil, nil, nil
}
