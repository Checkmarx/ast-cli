package iac_realtime

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

type Scanner struct {
	dockerManager *DockerManager
}

func NewScanner(dockerManager *DockerManager) *Scanner {
	return &Scanner{
		dockerManager: dockerManager,
	}
}

func (s *Scanner) RunScan(engine, volumeMap, tempDir, filePath string) ([]IacRealtimeResult, error) {
	err := s.dockerManager.RunKicsContainer(engine, volumeMap)
	return s.HandleScanResult(err, tempDir, filePath)
}

func (s *Scanner) HandleScanResult(err error, tempDir, filePath string) ([]IacRealtimeResult, error) {
	if err == nil {
		// No error, process results
		return s.processResults(tempDir, filePath)
	}

	msg := err.Error()
	code := s.extractErrorCode(msg)

	if slices.Contains(KicsErrorCodes, code) {
		return s.processResults(tempDir, filePath)
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == util.EngineNoRunningCode {
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

	data, err := io.ReadAll(file)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}
