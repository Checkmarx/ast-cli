package iac_realtime

import (
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type IacRealtimeService struct {
	JwtWrapper         wrappers.JWTWrapper
	FeatureFlagWrapper wrappers.FeatureFlagsWrapper
	fileHandler        *FileHandler
	dockerManager      *DockerManager
	scanner            *Scanner
}

func NewIacRealtimeService(jwt wrappers.JWTWrapper, flags wrappers.FeatureFlagsWrapper) *IacRealtimeService {
	fileHandler := NewFileHandler()
	dockerManager := NewDockerManager()
	scanner := NewScanner(dockerManager)

	return &IacRealtimeService{
		JwtWrapper:         jwt,
		FeatureFlagWrapper: flags,
		fileHandler:        fileHandler,
		dockerManager:      dockerManager,
		scanner:            scanner,
	}
}

func (svc *IacRealtimeService) RunIacRealtimeScan(filePath, engine, ignoredFilePath string) ([]IacRealtimeResult, error) {
	if err := svc.checkFeatureFlag(); err != nil {
		return nil, err
	}

	if err := svc.ensureLicense(); err != nil {
		return nil, err
	}

	if err := svc.validateFilePath(filePath); err != nil {
		return nil, err
	}

	svc.dockerManager.GenerateContainerID()

	volumeMap, tempDir, err := svc.fileHandler.PrepareScanEnvironment(filePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if tempDir != "" {
			_ = svc.fileHandler.CleanupTempDirectory(tempDir)
		}
	}()

	results, err := svc.scanner.RunScan(engine, volumeMap, tempDir, filePath)

	return results, err
}

func (svc *IacRealtimeService) checkFeatureFlag() error {
	enabled, err := realtimeengine.IsFeatureFlagEnabled(svc.FeatureFlagWrapper, wrappers.OssRealtimeEnabled)
	if err != nil || !enabled {
		return errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}
	return nil
}

func (svc *IacRealtimeService) ensureLicense() error {
	if err := realtimeengine.EnsureLicense(svc.JwtWrapper); err != nil {
		return errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}
	return nil
}

func (svc *IacRealtimeService) validateFilePath(filePath string) error {
	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		return errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}
	return nil
}
