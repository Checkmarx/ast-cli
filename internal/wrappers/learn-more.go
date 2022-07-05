package wrappers

type LearnMoreResponseModel struct {
	QueryId                uint64         `json:"queryId"`
	QueryName              string         `json:"queryName"`
	QueryDescriptionId     string         `json:"queryDescriptionId"`
	ResultDescription      string         `json:"resultDescription"'`
	Risk                   string         `json:"risk"`
	Cause                  string         `json:"cause"`
	GeneralRecommendations string         `json:"generalRecommendations"`
	Samples                []sampleObject `json:"samples"`
}

type sampleObject struct {
	ProgLanguage string `json:"progLanguage"`
	Code         string `json:"code"`
	Title        string `json:"title"`
}

type LearnMoreWrapper interface {
	GetLearnMoreDetails(string) (*LearnMoreResponseModel, *WebError, error)
}
