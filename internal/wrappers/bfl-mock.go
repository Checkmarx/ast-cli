package wrappers

type BFLMockWrapper struct{}

func (b *BFLMockWrapper) GetByScanID(scanID string, limit, offset uint64) (*BFLResponseModel, *ErrorModel, error) {
	return &BFLResponseModel{}, nil, nil
}
