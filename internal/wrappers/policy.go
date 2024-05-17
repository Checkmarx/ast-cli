package wrappers

type PolicyResponseModel struct {
	Status     string   `json:"status"`
	BreakBuild bool     `json:"breakBuild"`
	Polices    []Policy `json:"policies"`
}

type Policy struct {
	Name          string   `json:"policyName"`
	BreakBuild    bool     `json:"breakBuild"`
	Status        string   `json:"status"`
	Description   string   `json:"description"`
	RulesViolated []string `json:"rulesViolated"`
	Tags          []string `json:"tags"`
}

type PolicyWrapper interface {
	EvaluatePolicy(map[string]string) (*PolicyResponseModel, *WebError, error)
}
