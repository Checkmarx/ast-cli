package wrappers

type ApiModel struct {
	ApiName     string `json:"api_name"`
	Description string `json:"description"`
	ApiUrl      string `json:"api_url"`
	EngineName  string `json:"engine_name"`
	EngineId    string `json:"engine_id"`
}

var ApiModels = []ApiModel{
	{"GetAllSASTProject",
		" Gets all SAST projects",
		"https://cx_sast/projects",
		"SAST",
		"eSAST01"},
	{"GetAllSCAProject",
		"Gets all SCA projects",
		"https://cx_sca/projects",
		"SCA",
		"eSCA02"},
	{"Get all Iac Project",
		"This API gets all IaC projects",
		"https://cx_iac/projects",
		"Iac",
		"eIAC03"}}

type EnginesWrapper interface {
	GetAllAPIs(engineName string) ([]ApiModel, *ErrorModel, error)
}
