package wrappers

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *WebError, error)
	GetScanSummariesByScanIDS(params map[string]string) (*ScanSummariesModel, *WebError, error)
	GetAllResultsPackageByScanID(params map[string]string) (*[]ScaPackageCollection, *WebError, error)
	GetAllResultsTypeByScanID(params map[string]string) (*[]ScaTypeCollection, *WebError, error)
	GetResultsURL(projectID string) (string, error)
}
