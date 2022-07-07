package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type LearnMoreMockWrapper struct {
}

func (l LearnMoreMockWrapper) GetLearnMoreDetails(m map[string]string) (*[]*wrappers.LearnMoreResponse, *wrappers.WebError, error) {
	const mock = "MOCK"
	return &[]*wrappers.LearnMoreResponse{
		{
			QueryId:                mock,
			QueryName:              mock,
			QueryDescriptionId:     mock,
			ResultDescription:      mock,
			Risk:                   mock,
			Cause:                  mock,
			GeneralRecommendations: mock,
			Samples: []wrappers.SampleObject{
				{
					ProgLanguage: mock,
					Code:         mock,
					Title:        mock,
				},
			},
		},
	}, nil, nil
}
