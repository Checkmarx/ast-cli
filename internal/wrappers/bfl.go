package wrappers

type BFLWrapper interface {
	GetByScanID(scanID string, limit, offset uint64) (*BFLResponseModel, *ErrorModel, error)
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
