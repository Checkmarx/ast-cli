package wrappers

type PolicyResponseModel struct {
	Status     string   `json:"status"`
	BreakBuild bool     `json:"breakBuild"`
	Policies   []Policy `json:"policies"`
}

type Policy struct {
	Name          string   `json:"policyName"`
	BreakBuild    bool     `json:"breakBuild"`
	Status        string   `json:"status"`
	Description   string   `json:"description"`
	RulesViolated []string `json:"rulesViolated"`
	Tags          []string `json:"tags"`
}

type PrPolicy struct {
	Name       string   `json:"policyName"`
	RulesNames []string `json:"rulesNames"`
	BreakBuild bool     `json:"breakBuild"`
}

type PolicyWrapper interface {
	EvaluatePolicy(map[string]string) (*PolicyResponseModel, *WebError, error)
}
