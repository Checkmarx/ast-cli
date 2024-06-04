package services

import (
	"context"
	"strings"
	"sync"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"golang.org/x/sync/semaphore"
)

const (
	BatchSize = 200
)

func GetSastMetadataByIDs(sastMetaDataWrapper wrappers.SastMetadataWrapper, scanIDs []string) (*wrappers.SastMetadataModel, error) {
	totalBatches := (len(scanIDs) + BatchSize - 1) / BatchSize
	maxConcurrentGoRoutines := 10
	sem := semaphore.NewWeighted(int64(maxConcurrentGoRoutines))

	var wg sync.WaitGroup
	results := make(chan wrappers.SastMetadataModel, totalBatches)
	errors := make(chan error, totalBatches)
	ctx := context.Background()

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
		err := sem.Acquire(ctx, 1)
		if err != nil {
			return nil, err
		}
		go func() {
			defer wg.Done()
			defer sem.Release(1)

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
