package wrappers

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(scanID string, limit, offset uint64) (*ResultsResponseModel, *ResultError, error) {
	return &ResultsResponseModel{}, nil, nil
}
