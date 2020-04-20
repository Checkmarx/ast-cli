package wrappers

type BFLMockWrapper struct{}

func (b *BFLMockWrapper) GetByScanID(params map[string]string) (*BFLResponseModel, *ErrorModel, error) {
	const mock = "MOCK"
	return &BFLResponseModel{
		ID: mock,
		Trees: []BFLTreeModel{
			{
				ID: mock,
				BFL: ResultNode{
					Column:       0,
					FileName:     mock,
					FullName:     mock,
					Length:       0,
					Line:         0,
					MethodLine:   0,
					Name:         mock,
					NodeID:       0,
					DomType:      mock,
					NodeSystemID: mock,
				},
				Results: nil,
			},
		},
		TotalCount: 0,
	}, nil, nil
}
