package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type BflMockWrapper struct {
}

func (bfl *BflMockWrapper) GetBflByScanIDAndQueryID(params map[string]string) (
	*wrappers.BFLResponseModel,
	*wrappers.WebError,
	error,
) {
	const mock = "MOCK"
	return &wrappers.BFLResponseModel{
		ID: mock,
		Trees: []wrappers.BFLTreeModel{
			{
				ID: mock,
				BFL: &wrappers.ScanResultNode{
					Column:     0,
					FileName:   mock,
					FullName:   mock,
					Length:     0,
					Line:       0,
					MethodLine: 0,
					Name:       mock,
					DomType:    mock,
				},
				Results: nil,
			},
		},
		TotalCount: 1,
	}, nil, nil
}
