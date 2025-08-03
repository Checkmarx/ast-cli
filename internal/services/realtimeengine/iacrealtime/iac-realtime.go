package iacrealtime

import (
	"encoding/json"
	"fmt"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"os"
)

type IacRealtimeService struct {
	JwtWrapper         wrappers.JWTWrapper
	FeatureFlagWrapper wrappers.FeatureFlagsWrapper
	fileHandler        *FileHandler
	containerManager   IContainerManager
	scanner            *Scanner
}

func NewIacRealtimeService(jwt wrappers.JWTWrapper, flags wrappers.FeatureFlagsWrapper, containerManager IContainerManager) *IacRealtimeService {
	fileHandler := NewFileHandler()
	scanner := NewScanner(containerManager)

	return &IacRealtimeService{
		JwtWrapper:         jwt,
		FeatureFlagWrapper: flags,
		fileHandler:        fileHandler,
		containerManager:   containerManager,
		scanner:            scanner,
	}
}

func loadIgnoredIacFindings(path string) ([]IgnoredIacFinding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ignored []IgnoredIacFinding
	err = json.Unmarshal(data, &ignored)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func buildIgnoreMap(ignored []IgnoredIacFinding) map[string]bool {
	m := make(map[string]bool)
	for _, f := range ignored {
		key := fmt.Sprintf("%s_%s_%s", f.Title, f.FilePath, f.SimilarityID)
		m[key] = true
	}
	return m
}

func filterIgnoredFindings(results []IacRealtimeResult, ignoreMap map[string]bool) []IacRealtimeResult {
	filtered := make([]IacRealtimeResult, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s_%s_%s", r.Title, r.FilePath, r.SimilarityID)
		if !ignoreMap[key] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (svc *IacRealtimeService) RunIacRealtimeScan(filePath, engine, ignoredFilePath string) ([]IacRealtimeResult, error) {
	err := svc.runValidations(filePath)
	if err != nil {
		return nil, err
	}

	svc.containerManager.GenerateContainerID()

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
	if err != nil {
		return nil, err
	}

	if ignoredFilePath != "" {
		ignored, err := loadIgnoredIacFindings(ignoredFilePath)
		if err != nil {
			return nil, errorconstants.NewRealtimeEngineError("failed to load ignored IaC findings").Error()
		}
		ignoreMap := buildIgnoreMap(ignored)
		results = filterIgnoredFindings(results, ignoreMap)
	}

	return results, nil
}

func (svc *IacRealtimeService) runValidations(filePath string) error {
	if err := svc.checkFeatureFlag(); err != nil {
		return err
	}

	if err := svc.ensureLicense(); err != nil {
		return err
	}

	if err := svc.validateFilePath(filePath); err != nil {
		return err
	}
	return nil
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
