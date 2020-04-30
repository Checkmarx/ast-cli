package wrappers

type BFLWrapper interface {
	GetByScanID(params map[string]string) (*BFLResponseModel, *ErrorModel, error)
}

type BFLResponseModel struct {
	ID         string
	Trees      []BFLTreeModel
	TotalCount int
}
type BFLTreeModel struct {
	ID      string
	BFL     ResultNode
	Results []ResultResponseModel
}
