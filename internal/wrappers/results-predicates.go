package wrappers

type Predicate struct {
	SimilarityId string   `json:"similarityId"`
	ProjectId	string   `json:"projectId"`
	State		string   `json:"state"`
	Comment		string   `json:"comment"`
	Severity    string	 `json:"severity"`
	ScannerType string   `json:"scannerType"`
}


type ResultsPredicatesWrapper interface {
	PredicateSeverityAndStateForSAST(predicate *Predicate)(error)
	PredicateSeverityAndStateForKICS(predicate *Predicate)(error)
}
