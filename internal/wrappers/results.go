package wrappers

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *WebError, error)
	GetResultPlus(id string) (*ScanResultsCollection, *WebError, error)
}
