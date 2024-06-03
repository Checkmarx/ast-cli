package services

import (
	"slices"
	"strings"
	"sync"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const (
	BatchSize = 200
)

type ResultWithSequence struct {
	Sequence int
	Model    *wrappers.SastMetadataModel
}

func GetSastMetadataByIDs(sastMetaDataWrapper wrappers.SastMetadataWrapper, scanIDs []string) (*wrappers.SastMetadataModel, error) {
	totalBatches := (len(scanIDs) + BatchSize - 1) / BatchSize
	maxConcurrentGoroutines := 10
	semaphore := make(chan struct{}, maxConcurrentGoroutines)

	var wg sync.WaitGroup
	results := make(chan ResultWithSequence, totalBatches)
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
		go func(seq int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			result, err := sastMetaDataWrapper.GetSastMetadataByIDs(batchParams)
			if err != nil {
				errors <- err
				return
			}
			results <- ResultWithSequence{Sequence: seq, Model: result}
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	if len(errors) > 0 {
		return nil, <-errors
	}

	var sortedResults []ResultWithSequence
	for result := range results {
		sortedResults = append(sortedResults, result)
	}
	// sort results by sequence - we need to keep the order of the scans as they were requested
	sortedResults = sortResults(sortedResults)

	return makeSastMetadataModelFromResults(sortedResults), nil
}

func sortResults(results []ResultWithSequence) []ResultWithSequence {
	slices.SortFunc(results, func(a, b ResultWithSequence) int {
		if a.Sequence < b.Sequence {
			return -1
		}
		if a.Sequence > b.Sequence {
			return 1
		}
		return 0
	})
	return results
}

func makeSastMetadataModelFromResults(results []ResultWithSequence) *wrappers.SastMetadataModel {
	finalResult := &wrappers.SastMetadataModel{}
	for _, result := range results {
		finalResult.TotalCount += result.Model.TotalCount
		finalResult.Scans = append(finalResult.Scans, result.Model.Scans...)
		finalResult.Missing = append(finalResult.Missing, result.Model.Missing...)
	}
	return finalResult
}
