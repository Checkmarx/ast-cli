package wrappers

type EngineListResponseModel struct {
	Engines []EngineList `json:"engines"`
}
type EngineList struct {
	EngineId   string       `json:"engine_id"`   // [git|upload]
	EngineName string       `json:"engine_name"` // One of [GitProjectHandler|ScanHandler]
	APIs       []EngineAPIs `json:"apis"`
}

type EngineAPIs struct {
	ApiURL      string `json:"api-url"`  // [git|upload]
	ApiName     string `json:"api-name"` // One of [GitProjectHandler|ScanHandler]
	Description string `json:"api-description"`
}

type EnginesWrapper interface {
	Get(engineName string) *EngineListResponseModel
}
