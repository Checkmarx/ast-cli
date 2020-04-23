package wrappers

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(scanID string,
	limit, offset uint64) ([]ResultResponseModel, *ResultError, error) {
	return []ResultResponseModel{}, nil, nil
}
