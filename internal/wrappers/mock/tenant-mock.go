package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

var TenantConfiguration []*wrappers.TenantConfigurationResponse

type TenantConfigurationMockWrapper struct {
}

func (t TenantConfigurationMockWrapper) GetTenantConfiguration() (
	*[]*wrappers.TenantConfigurationResponse,
	*wrappers.WebError,
	error,
) {
	if len(TenantConfiguration) == 0 {
		TenantConfiguration = []*wrappers.TenantConfigurationResponse{
			{
				Key:   "scan.config.plugins.ideScans",
				Value: "true",
			},
			{
				Key:   "scan.config.plugins.aiGuidedRemediation",
				Value: "true",
			},
		}
	}
	return &TenantConfiguration, nil, nil
}
