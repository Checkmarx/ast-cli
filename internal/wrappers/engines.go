package wrappers

const ()

type EnginesAPIInventoryResponseModel struct {
	Engines []EnginesAPIResponseModel `json:"engines"`
}

type EnginesAPIResponseModel struct {
	Engine_Id   string           `json:"engine_id"`
	Engine_Name string           `json:"engine_name"`
	Apis        []EnginesAPIData `json:"apis"`
}

type EnginesAPIData struct {
	Api_Url     string `json:"api_url"`
	Api_Name    string `json:"api_name"`
	Description string `json:"description"`
}

type EnginesWrapper interface {
	Get(engineName string) (*EnginesAPIInventoryResponseModel, *ErrorModel, error)
}
