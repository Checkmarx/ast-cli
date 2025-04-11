package wrappers

type ApiModel struct {
	ApiName     string `json:"api_name"`
	Description string `json:"description"`
	ApiUrl      string `json:"api_url"`
	EngineName  string `json:"engine_name"`
	EngineId    string `json:"engine_id"`
}

var ApiModels = []ApiModel{
	{"GetSASTProject1",
		"This endpoint related to fetch projects",
		"https://cx_sast/projects",
		"SAST",
		"eSAST01",
	},

	{"GetSCAProject1",
		"This endpoint related to fetch projects",
		"https://cx_sca/projects",
		"SCA",
		"eSCA1",
	},

	{"GetIacProject1",
		"This endpoint related to fetch projects",
		"https://cx_iac/projects",
		"Iac",
		"eIac01",
	},
}

type EnginesWrapper interface {
	GetAllAPIs(engineName string) ([]ApiModel, *ErrorModel, error)
}
