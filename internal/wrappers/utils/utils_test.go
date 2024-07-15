package utils

import (
	"log"
	"testing"
)

func TestCleanURL_CleansCorrectly(t *testing.T) {
	uri := "https://codebashing.checkmarx.com/courses/java/////lessons/sql_injection/////"
	want := "https://codebashing.checkmarx.com/courses/java/lessons/sql_injection"
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != false {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, false)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}

func TestCleanURL_invalid_URL_escape_error(t *testing.T) {
	uri := "#)@($_(*#_(*@$_))%(_#@_+#@$)$_$#@_@_##}^^^}!)(()!#@(`SPPSCOK^Ç^Ç`P$_$"
	want := ""
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != true {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, true)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}

func TestCleanURL_cleans_correctly2(t *testing.T) {
	uri := "http://localhost:42/////test//test"
	want := "http://localhost:42/test/test"
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != false {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, false)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}

func TestContains(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		str      string
		expected bool
	}{
		{
			name:     "String present in slice",
			input:    []string{"apple", "banana", "orange"},
			str:      "banana",
			expected: true,
		},
		{
			name:     "String not present in slice",
			input:    []string{"apple", "banana", "orange"},
			str:      "grape",
			expected: false,
		},
		{
			name:     "Empty slice",
			input:    []string{},
			str:      "test",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    []string{"apple", "banana", "orange"},
			str:      "",
			expected: false,
		},
		{
			name:     "String present multiple times",
			input:    []string{"apple", "banana", "orange", "banana"},
			str:      "banana",
			expected: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Create a local variable with the same name
		t.Run(tc.name, func(t *testing.T) {
			result := Contains(tc.input, tc.str)
			if result != tc.expected {
				t.Errorf("Expected %v but got %v for input %v and string %s", tc.expected, result, tc.input, tc.str)
			}
		})
	}
}
