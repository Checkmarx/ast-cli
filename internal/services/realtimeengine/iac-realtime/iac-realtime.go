package iac_realtime

import "github.com/checkmarx/ast-cli/internal/wrappers"

type IacRealtimeService struct {
	JwtWrapper         wrappers.JWTWrapper
	FeatureFlagWrapper wrappers.FeatureFlagsWrapper
}

func NewIacRealtimeService(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
) *IacRealtimeService {
	return &IacRealtimeService{
		JwtWrapper:         jwtWrapper,
		FeatureFlagWrapper: featureFlagWrapper,
	}
}

func (o *IacRealtimeService) RunIacRealtimeScan(filePath, ignoredFilePath string) (results *IacRealtimeResults, err error) {
	return nil, nil
}
