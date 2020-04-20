package wrappers

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(scanID string, limit, offset uint64) (*ResultsResponseModel, *ErrorModel, error) {
	return &ResultsResponseModel{
		Results: []ResultResponseModel{
			{
				QueryID:                         0,
				QueryName:                       scanID,
				Severity:                        scanID,
				PathSystemID:                    scanID,
				PathSystemIDBySimiAndFilesPaths: scanID,
				ID:                              scanID,
				FirstScanID:                     scanID,
				FirstFoundAt:                    scanID,
				FoundAt:                         scanID,
				Status:                          scanID,
			},
		},
		TotalCount: 1,
	}, nil, nil
}
