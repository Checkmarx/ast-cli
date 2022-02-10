package wrappers

type BFLResponseModel struct {
	ID         string         `json:"id"`
	Trees      []BFLTreeModel `json:"trees"`
	TotalCount int            `json:"totalCount"`
}

type BFLTreeModel struct {
	ID                  string                     `json:"id"`
	BFL                 *ScanResultNode            `json:"bfl"`
	Results             []*ScanResultData          `json:"results"`
	Nodes               map[string]*ScanResultNode `json:"nodes,omitempty"`
	NodesAdjacencyPairs [][]string                 `json:"nodesAdjacencyPairs,omitempty"`
}

type BflWrapper interface {
	GetBflByScanIDAndQueryID(params map[string]string) (*BFLResponseModel, *WebError, error)
}
