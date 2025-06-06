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

type AzurePRModel struct {
	ScanID    string     `json:"scanId"`
	ScmToken  string     `json:"scmToken"`
	Namespace string     `json:"namespace"`
	PrNumber  int        `json:"prNumber"`
	Policies  []PrPolicy `json:"violatedPolicyList"`
	APIURL    string     `json:"apiUrl"`
}

type BitbucketCloudPRModel struct {
	ScanID    string     `json:"scanId"`
	ScmToken  string     `json:"scmToken"`
	Namespace string     `json:"namespace"`
	RepoName  string     `json:"repoName"`
	PRID      int        `json:"prId"`
	Policies  []PrPolicy `json:"violatedPolicyList"`
}

type BitbucketServerPRModel struct {
	ScanID     string     `json:"scanId"`
	ScmToken   string     `json:"scmToken"`
	ServerURL  string     `json:"apiUrl"`
	ProjectKey string     `json:"namespace"`
	RepoName   string     `json:"repoName"`
	PRID       int        `json:"prNumber"`
	Policies   []PrPolicy `json:"violatedPolicyList"`
}
type PRWrapper interface {
	PostPRDecoration(model interface{}) (string, *WebError, error)
}
