package iacrealtime

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

// Scanner handles the KICS scan process
type Scanner struct {
	dockerManager IContainerManager
	mapper        *Mapper
}

func NewScanner(dockerManager IContainerManager) *Scanner {
	return &Scanner{
		dockerManager: dockerManager,
		mapper:        NewMapper(),
	}
}

func (s *Scanner) RunScan(engine, volumeMap, tempDir, filePath string) ([]IacRealtimeResult, error) {
	kicsErrCode := s.dockerManager.RunKicsContainer(engine, volumeMap)
	return s.HandleScanResult(kicsErrCode, tempDir, filePath)
}

func (s *Scanner) HandleScanResult(kicsErrorCode error, tempDir, filePath string) ([]IacRealtimeResult, error) {
	if kicsErrorCode == nil {
		// No error, process results
		return s.processResults(tempDir, filePath)
	}

	msg := kicsErrorCode.Error()
	code := s.extractErrorCode(msg)

	if slices.Contains(KicsErrorCodes, code) {
		return s.processResults(tempDir, filePath)
	}

	if exitErr, ok := kicsErrorCode.(*exec.ExitError); ok && exitErr.ExitCode() == util.EngineNoRunningCode {
		return nil, errors.New(util.NotRunningEngineMessage)
	}

	if strings.Contains(msg, util.InvalidEngineError) || strings.Contains(msg, util.InvalidEngineErrorWindows) {
		return nil, errors.New(util.InvalidEngineMessage)
	}

	return nil, errors.Errorf("Check container engine state. Failed: %s", msg)
}

func (s *Scanner) processResults(tempDir, filePath string) ([]IacRealtimeResult, error) {
	results, err := s.readKicsResultsFile(tempDir)
	if err != nil {
		return nil, errors.Errorf("failed to read KICS results: %s", err)
	}

	mapper := NewMapper()
	iacRealtimeResults := mapper.ConvertKicsToIacResults(&results, filePath)

	return iacRealtimeResults, nil
}

func (s *Scanner) extractErrorCode(msg string) string {
	if idx := strings.LastIndex(msg, " "); idx != -1 {
		return msg[idx+1:]
	}
	return ""
}

const maxJSONFileSize = 50 * 1024 * 1024 // 50MB limit for JSON files

func (s *Scanner) readKicsResultsFile(tempDir string) (wrappers.KicsResultsCollection, error) {
	var result wrappers.KicsResultsCollection
	path := filepath.Join(tempDir, ContainerResultsFileName)

	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer func() {
		_ = file.Close()
	}()

	// Limit file size to prevent unsafe deserialization
	fileInfo, err := file.Stat()
	if err != nil {
		return result, err
	}

	if fileInfo.Size() > maxJSONFileSize {
		return result, errors.New("JSON file too large for safe deserialization")
	}

	// Use a limited reader for additional safety
	limitedReader := io.LimitReader(file, maxJSONFileSize)
	decoder := json.NewDecoder(limitedReader)

	if err := decoder.Decode(&result); err != nil {
		return result, err
	}

	return result, nil
}
