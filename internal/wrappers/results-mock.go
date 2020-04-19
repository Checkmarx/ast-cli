package wrappers

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(scanID string, limit, offset uint64) (*ResultsResponseModel, *ErrorModel, error) {
	return &ResultsResponseModel{}, nil, nil
}
