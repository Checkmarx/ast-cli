package wrappers

type EnginesCollectionResponseModel struct {
	Engines []EngineResponseModel `json:"engines"`
}

type EngineAPIModel struct {
	ApiURL      string `json:"api-url"`
	ApiName     string `json:"api-name"`
	Description string `json:"description"`
}
type EngineResponseModel struct {
	EngineID   string           `json:"engine_id"`
	EngineName string           `json:"engine_name"`
	Apis       []EngineAPIModel `json:"apis"`
}

type EngineWrapper interface {
	Get(engineName string) (*EnginesCollectionResponseModel, error)
}
