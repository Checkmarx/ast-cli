package wrappers

type CodeBashingWrapper interface {
	GetCodeBashingLinks(params map[string]string) (*[]CodeBashingCollection, *WebError, error)
}
