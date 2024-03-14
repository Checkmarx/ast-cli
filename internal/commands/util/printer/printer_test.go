package printer

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestPrintInvalidFormat(t *testing.T) {
	err := Print(os.Stdout, nil, "invalid_format")
	assert.Assert(t, err != nil, "An error should have been thrown due the wrong format")
}

func TestPrintJson(t *testing.T) {
	// valid json to marshal
	err := Print(os.Stdout, "{\"jsonTag\": \"jsonValue\"", FormatJSON)
	assert.NilError(t, err, "json print must run well")

	// invalid json to marshal
	err = Print(os.Stdout, make(chan int), FormatJSON)
	assert.Assert(t, err != nil, "An error should have been thrown due the invalid json format")
	fmt.Println(err.Error())
	assert.Assert(t, err.Error() == "json: unsupported type: chan int")
}

func TestPrintList(t *testing.T) {
	err := Print(os.Stdout, nil, FormatList)
	assert.NilError(t, err, "list print must run well")

	err = Print(os.Stdout, []string{}, FormatList)
	assert.NilError(t, err, "list print must run well")
}

func TestPrintTable(t *testing.T) {
	// Test null table
	err := Print(os.Stdout, nil, FormatTable)
	assert.NilError(t, err, "table print must run well")

	// Test empty table
	err = Print(os.Stdout, []string{}, FormatTable)
	assert.NilError(t, err, "table print must run well")

	// Test empty table
	err = Print(os.Stdout, []string{"column1", "column2", "column3"}, FormatTable)
	assert.NilError(t, err, "table print must run well")
}

// test is format
func TestIsFormat(t *testing.T) {
	assert.Assert(t, IsFormat("json", FormatJSON), "json is a valid format")
	assert.Assert(t, IsFormat("JSON", FormatJSON), "JSON is a valid format")
	assert.Assert(t, IsFormat("list", FormatList), "list is a valid format")
	assert.Assert(t, IsFormat("table", FormatTable), "table is a valid format")
	assert.Assert(t, !IsFormat("invalid_format", FormatTable), "invalid_format is not a valid format")
}

func TestColumnReformat(t *testing.T) {
	// Test nil
	columnReformat(nil)

	// Test empty
	columnReformat([]*entity{})
}

func TestParseName(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedResult string
	}{
		{
			name:           "Valid name",
			input:          "name:key",
			expectedResult: "key",
		},
		{
			name:           "Empty name",
			input:          "name:",
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a function using the parseName function with the test input
			f := parseName(tc.input)

			// Create a property to pass to the function
			prop := &property{}

			// Call the generated function
			f(prop, nil)

			// Check if the key was set correctly
			if prop.Key != tc.expectedResult {
				t.Errorf("Expected key %s, but got %s", tc.expectedResult, prop.Key)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		raw            interface{}
		expectedResult string
	}{
		{
			name:           "Valid time with format",
			input:          "time:2006-01-02",
			raw:            time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			expectedResult: "2024-03-14",
		},
		{
			name:           "Nil time pointer",
			input:          "time:15:04:05",
			raw:            (*time.Time)(nil),
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseTime(tc.input)

			prop := &property{}

			f(prop, tc.raw)

			if prop.Value != tc.expectedResult {
				t.Errorf("Expected value %s, but got %s", tc.expectedResult, prop.Value)
			}
		})
	}
}

func TestParseMaxlen(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		raw            string
		expectedResult string
	}{
		{
			name:           "Valid maxlen",
			input:          "maxlen:5",
			raw:            "123456789",
			expectedResult: "12345",
		},
		{
			name:           "Value shorter than maxlen",
			input:          "maxlen:10",
			raw:            "12345",
			expectedResult: "12345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a function using the parseMaxlen function with the test input
			f := parseMaxlen(tc.input)

			// Create a property to pass to the function
			prop := &property{Value: tc.raw}

			// Call the generated function
			f(prop, nil)

			// Check if the value was truncated correctly
			if prop.Value != tc.expectedResult {
				t.Errorf("Expected value %s, but got %s", tc.expectedResult, prop.Value)
			}
		})
	}
}

func TestGetFormatter(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		raw            interface{}
		expectedResult string
	}{
		{
			name:           "Maxlen formatter",
			input:          "maxlen:5",
			raw:            "123456789",
			expectedResult: "12345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the formatter function using the getFormatter function with the test input
			f := getFormatter(tc.input)

			// Perform type assertion to ensure raw is treated as a string
			rawStr, ok := tc.raw.(string)
			if !ok {
				t.Fatalf("Unable to assert %v as string", tc.raw)
			}

			// Create a property to pass to the function
			prop := &property{Value: rawStr}

			// Call the generated function
			f(prop, rawStr)

			// Check if the property's value was formatted correctly
			if prop.Value != tc.expectedResult {
				t.Errorf("Expected value %s, but got %s", tc.expectedResult, prop.Value)
			}
		})
	}
}
