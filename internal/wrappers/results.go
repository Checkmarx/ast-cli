package wrappers

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *WebError, error)
}
