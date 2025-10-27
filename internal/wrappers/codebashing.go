package wrappers

type CodeBashingWrapper interface {
	GetCodeBashingLinks(queryId string, codeBashingURL string) (*[]CodeBashingCollection, *WebError, error)
	GetCodeBashingURL(field string) (string, error)
	BuildCodeBashingParams([]CodeBashingParamsCollection) (map[string]string, error)
}
