package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type TenantConfigurationMockWrapper struct {
}

func (t TenantConfigurationMockWrapper) GetTenantConfiguration() (
	*[]*wrappers.TenantConfigurationResponse,
	*wrappers.WebError,
	error,
) {
	return &[]*wrappers.TenantConfigurationResponse{
		{
			Key:   "scan.config.plugins.ideScans",
			Value: "true",
		},
	}, nil, nil
}
