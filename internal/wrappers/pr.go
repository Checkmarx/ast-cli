package wrappers

type PRResponseModel struct {
	Message string
}

type PRModel struct {
	ScanID    string     `json:"scanId"`
	ScmToken  string     `json:"scmToken"`
	Namespace string     `json:"namespace"`
	RepoName  string     `json:"repoName"`
	PrNumber  int        `json:"prNumber"`
	Policies  []PrPolicy `json:"violatedPolicyList"`
	APIURL    string     `json:"apiUrl"`
}

type GitlabPRModel struct {
	ScanID          string     `json:"scanId"`
	ScmToken        string     `json:"scmToken"`
	Namespace       string     `json:"namespace"`
	RepoName        string     `json:"repoName"`
	IiD             int        `json:"iid"`
	GitlabProjectID int        `json:"gitlabProjectID"`
	Policies        []PrPolicy `json:"violatedPolicyList"`
	APIURL          string     `json:"apiUrl"`
}

type PRWrapper interface {
	PostPRDecoration(model *PRModel) (string, *WebError, error)
	PostGitlabPRDecoration(model *GitlabPRModel) (string, *WebError, error)
}
