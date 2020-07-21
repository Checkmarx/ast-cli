package wrappers

import (
	"github.com/checkmarxDev/sast-results/pkg/bfl"
	resultsReader "github.com/checkmarxDev/sast-results/pkg/reader"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsBfl "github.com/checkmarxDev/sast-results/pkg/web/path/bfl"
)

type BFLMockWrapper struct{}

func (b *BFLMockWrapper) GetByScanID(_ map[string]string) (*resultsBfl.Forest, *resultsHelpers.WebError, error) {
	const mock = "MOCK"
	return &resultsBfl.Forest{
		ID: mock,
		Trees: []*bfl.BFL{
			{
				ID: mock,
				BflNode: &resultsReader.ResultNode{
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
