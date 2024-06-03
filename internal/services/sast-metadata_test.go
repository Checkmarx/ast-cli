package services

import (
	"reflect"
	"testing"
)

func TestGetSastMetadataByIDs(t *testing.T) {

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
