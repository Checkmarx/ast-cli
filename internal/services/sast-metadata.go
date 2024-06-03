package services

import (
	"strings"
	"sync"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	BatchSize = 200
)

func GetSastMetadataByIDs(sastMetaDataWrapper wrappers.SastMetadataWrapper, scanIDs []string) (*wrappers.SastMetadataModel, error) {
	totalBatches := (len(scanIDs) + BatchSize - 1) / BatchSize
	maxConcurrentGoroutines := 10
	semaphore := make(chan struct{}, maxConcurrentGoroutines)

	var wg sync.WaitGroup
	results := make(chan wrappers.SastMetadataModel, totalBatches)
	errors := make(chan error, totalBatches)

	for i := 0; i < totalBatches; i++ {
		start := i * BatchSize
		end := start + BatchSize
		if end > len(scanIDs) {
			end = len(scanIDs)
		}

		batchParams := map[string]string{
			commonParams.ScanIDsQueryParam: strings.Join(scanIDs[start:end], ","),
		}

		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			result, err := sastMetaDataWrapper.GetSastMetadataByIDs(batchParams)
			if err != nil {
				errors <- err
				return
			}
			results <- *result
		}()
	}

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	if len(errors) > 0 {
		return nil, <-errors
	}

	var models []wrappers.SastMetadataModel
	for result := range results {
		models = append(models, result)
	}
	return makeSastMetadataModelFromResults(models), nil
}

func makeSastMetadataModelFromResults(results []wrappers.SastMetadataModel) *wrappers.SastMetadataModel {
	finalResult := &wrappers.SastMetadataModel{}
	for _, result := range results {
		finalResult.TotalCount += result.TotalCount
		finalResult.Scans = append(finalResult.Scans, result.Scans...)
		finalResult.Missing = append(finalResult.Missing, result.Missing...)
	}
	return finalResult
}
