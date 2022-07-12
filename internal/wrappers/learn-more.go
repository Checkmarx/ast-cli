package wrappers

type LearnMoreResponse struct {
	QueryID                string         `json:"queryId"`
	QueryName              string         `json:"queryName"`
	QueryDescriptionID     string         `json:"queryDescriptionId"`
	ResultDescription      string         `json:"resultDescription"`
	Risk                   string         `json:"risk"`
	Cause                  string         `json:"cause"`
	GeneralRecommendations string         `json:"generalRecommendations"`
	Samples                []SampleObject `json:"samples"`
}

type SampleObject struct {
	ProgLanguage string `json:"progLanguage"`
	Code         string `json:"code"`
	Title        string `json:"title"`
}

type LearnMoreWrapper interface {
	GetLearnMoreDetails(map[string]string) (*[]*LearnMoreResponse, *WebError, error)
}
