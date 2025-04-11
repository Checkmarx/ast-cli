package wrappers

type EnginesHTTPWrapper struct {
}

func NewHTTPEnginesWrapper() EnginesWrapper {
	return &EnginesHTTPWrapper{}
}

func (e *EnginesHTTPWrapper) GetAllAPIs(engineName string) ([]ApiModel, *ErrorModel, error) {
	var filteredApiModels []ApiModel = FilterByEngineType(ApiModels, engineName)
	return filteredApiModels, nil, nil
}

func FilterByEngineType(apiModel []ApiModel, engineName string) []ApiModel {
	var filtered []ApiModel

	if engineName == "" {
		return apiModel
	}
	for _, apiM := range apiModel {
		if apiM.EngineName == engineName {
			filtered = append(filtered, apiM)
		}
	}
	return filtered
}
