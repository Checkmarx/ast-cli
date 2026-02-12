package utils

import (
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ascaLocationKey = "asca-location"
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
func TestLimitSlice(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		limit    int
		expected []int
	}{
		{
			name:     "Limit greater than slice length",
			input:    []int{1, 2, 3, 4, 5},
			limit:    10,
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Limit less than slice length",
			input:    []int{1, 2, 3, 4, 5},
			limit:    3,
			expected: []int{1, 2, 3},
		},
		{
			name:     "Limit equal to zero",
			input:    []int{1, 2, 3, 4, 5},
			limit:    0,
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Limit equal to slice length",
			input:    []int{1, 2, 3, 4, 5},
			limit:    5,
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Empty slice",
			input:    []int{},
			limit:    3,
			expected: []int{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := LimitSlice(tc.input, tc.limit)
			if !equal(result, tc.expected) {
				t.Errorf("Expected %v but got %v for input %v and limit %d", tc.expected, result, tc.input, tc.limit)
			}
		})
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func clearOptionalParams() {
	mutex.Lock()
	optionalParams = make(map[string]string)
	mutex.Unlock()
}

func TestSetAndGetAllowedKey(t *testing.T) {
	clearOptionalParams()
	key := ascaLocationKey
	value := "location-1"

	err := SetOptionalParam(key, value)
	assert.Nil(t, err)

	if !hasOptionalParam(key) {
		t.Fatalf("expected hasOptionalParam(%q) to be true", key)
	}
	got := GetOptionalParam(key)
	if got != value {
		t.Fatalf("expected GetOptionalParam(%q) == %q, got %q", key, value, got)
	}
}

func TestSetNotAllowedKeyDoesNotSet(t *testing.T) {
	clearOptionalParams()

	key := "not-allowed-key"
	value := "value"

	err := SetOptionalParam(key, value)
	assert.Nil(t, err)

	if hasOptionalParam(key) {
		t.Fatalf("expected hasOptionalParam(%q) to be false for disallowed key", key)
	}

	got := GetOptionalParam(key)
	if got != "" {
		t.Fatalf("expected GetOptionalParam(%q) == empty, got %q", key, got)
	}
}

func TestSetWithTrimmedKeyBehavior(t *testing.T) {
	clearOptionalParams()

	originalKey := "  asca-location  "
	trimmedKey := ascaLocationKey
	value := "trimmed-value"

	err := SetOptionalParam(originalKey, value)
	assert.Nil(t, err)

	if got := GetOptionalParam(originalKey); got != value {
		t.Fatalf("expected GetOptionalParam(%q) == %q, got %q", originalKey, value, got)
	}

	if got := GetOptionalParam(trimmedKey); got != value {
		t.Fatalf("expected GetOptionalParam(%q) == %q, got %q", originalKey, value, got)
	}
}

func TestConcurrentSetOptionalParam(t *testing.T) {
	clearOptionalParams()
	var wg sync.WaitGroup

	// spawn several goroutines setting allowed and disallowed keys
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := SetOptionalParam(ascaLocationKey, "v")
			assert.Nil(t, err)
			err = SetOptionalParam("not-allowed", "x")
			assert.NotNil(t, err)
		}()
	}
	wg.Wait()

	if !hasOptionalParam(ascaLocationKey) {
		t.Fatalf("expected hasOptionalParam(%q) to be true after concurrent sets", "asca-location")
	}
	if GetOptionalParam("asca-location") != "v" {
		t.Fatalf("expected GetOptionalParam(%q) == %q", ascaLocationKey, "v")
	}
}
