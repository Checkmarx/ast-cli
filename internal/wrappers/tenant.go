package wrappers

type TenantConfigurationResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TenantConfigurationWrapper interface {
	GetTenantConfiguration() (*[]*TenantConfigurationResponse, *WebError, error)
}
