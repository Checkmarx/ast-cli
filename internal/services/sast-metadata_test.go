package services

import (
	"fmt"
	"reflect"
	"testing"

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
			for i, scan := range actual.Scans {
				expectedScanID := fmt.Sprintf("ConcurrentTest%d", i)
				if scan.ScanID != expectedScanID {
					t.Errorf("GetSastMetadataByIDs(%v) returned scan with ScanID %s, expected %s", tt.scanIDs, scan.ScanID, expectedScanID)
				}
			}
			assert.Equal(t, len(tt.scanIDs), len(actual.Scans))
		})
	}
}
func TestSortResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []ResultWithSequence
		expected []ResultWithSequence
	}{
		{
			name: "Already sorted",
			input: []ResultWithSequence{
				{Sequence: 1},
				{Sequence: 2},
				{Sequence: 3},
			},
			expected: []ResultWithSequence{
				{Sequence: 1},
				{Sequence: 2},
				{Sequence: 3},
			},
		},
		{
			name: "Reverse order",
			input: []ResultWithSequence{
				{Sequence: 3},
				{Sequence: 2},
				{Sequence: 1},
			},
			expected: []ResultWithSequence{
				{Sequence: 1},
				{Sequence: 2},
				{Sequence: 3},
			},
		},
		{
			name: "Random order",
			input: []ResultWithSequence{
				{Sequence: 2},
				{Sequence: 3},
				{Sequence: 1},
			},
			expected: []ResultWithSequence{
				{Sequence: 1},
				{Sequence: 2},
				{Sequence: 3},
			},
		},
		{
			name: "Duplicate sequences",
			input: []ResultWithSequence{
				{Sequence: 2},
				{Sequence: 1},
				{Sequence: 2},
			},
			expected: []ResultWithSequence{
				{Sequence: 1},
				{Sequence: 2},
				{Sequence: 2},
			},
		},
		{
			name:     "Empty slice",
			input:    []ResultWithSequence{},
			expected: []ResultWithSequence{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual := sortResults(tt.input)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("sortResults(%v) = %v, want %v", tt.input, actual, tt.expected)
			}
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
