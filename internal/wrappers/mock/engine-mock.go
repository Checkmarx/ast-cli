package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type NewHTTPEnginesMockWrapper struct{}

func (n *NewHTTPEnginesMockWrapper) GetAllAPIs(engineType string) ([]wrappers.ApiModel, *wrappers.ErrorModel, error) {
	var filteredApiModels []wrappers.ApiModel = wrappers.FilterByEngineType(wrappers.ApiModels, engineType)
	return filteredApiModels, nil, nil
}
