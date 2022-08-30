package wrappers

type PRResponseModel struct {
	Message string
}

type PRModel struct {
	ScanID    string `json:"scanId"`
	ScmToken  string `json:"scmToken"`
	Namespace string `json:"namespace"`
	RepoName  string `json:"repoName"`
	PrNumber  int    `json:"prNumber"`
}

type PRWrapper interface {
	PostPRDecoration(model *PRModel) (string, *WebError, error)
}
