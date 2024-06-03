package services

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert" //nolint:depguard
)

func TestGetSastMetadataByIDs(t *testing.T) {
	tests := []struct {
		name    string
		scanIDs []string
	}{
		{
			name:    "Multiple batches",
			scanIDs: createScanIDs(2000),
		},
		{
			name:    "Single batch",
			scanIDs: createScanIDs(100),
		},
		{
			name:    "Empty slice",
			scanIDs: []string{},
		},
		{
			name:    "Multiple batches with partial last batch",
			scanIDs: createScanIDs(893),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			wrapper := &mock.SastMetadataMockWrapper{}
			actual, err := GetSastMetadataByIDs(wrapper, tt.scanIDs)
			if err != nil {
				t.Errorf("GetSastMetadataByIDs(%v) returned an error: %v", tt.scanIDs, err)
			}
			assert.True(t, checkAllScanExists(actual.Scans, "ConcurrentTest", len(tt.scanIDs)))
			assert.Equal(t, len(tt.scanIDs), len(actual.Scans))
		})
	}
}

func createScanIDs(count int) []string {
	scanIDs := make([]string, count)
	for i := 0; i < count; i++ {
		scanIDs[i] = fmt.Sprintf("ConcurrentTest%d", i)
	}
	return scanIDs
}

func checkAllScanExists(scans []wrappers.Scans, prefix string, count int) bool {
	existingScans := make(map[int]bool)

	for i := range scans {
		scan := &scans[i]
		if strings.HasPrefix(scan.ScanID, prefix) {
			scanNumberStr := scan.ScanID[len(prefix):]
			if scanNumber, err := strconv.Atoi(scanNumberStr); err == nil {
				existingScans[scanNumber] = true
			}
		}
	}

	for i := 0; i < count; i++ {
		if !existingScans[i] {
			return false
		}
	}
	return true
}
