package wrappers

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(params map[string]string) (*ResultsResponseModel, *ErrorModel, error) {
	const mock = "MOCK"
	return &ResultsResponseModel{
		Results: []ResultResponseModel{
			{
				QueryID:                         0,
				QueryName:                       mock,
				Severity:                        mock,
				PathSystemID:                    mock,
				PathSystemIDBySimiAndFilesPaths: mock,
				ID:                              mock,
				FirstScanID:                     mock,
				FirstFoundAt:                    mock,
				FoundAt:                         mock,
				Status:                          mock,
			},
		},
		TotalCount: 1,
	}, nil, nil
}
