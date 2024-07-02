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

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		name          string
		formatName    string
		expectedFunc  func(*property, interface{})
		expectedPanic bool
		property      property
	}{
		{"Valid maxlen format", "maxlen:5", parseMaxlen("maxlen:5"), false, property{Key: "key", Value: "1234567890"}},
		{"Valid time format", "time:2006-01-02", parseTime("time:2006-01-02"), false, property{Key: "key", Value: "2021-09-01"}},
		{"Valid name format", "name:CustomName", parseName("name:CustomName"), false, property{Key: "key", Value: "value"}},
		{"Invalid format", "invalid_format", nil, true, property{}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != test.expectedPanic {
					t.Errorf("Expected panic: %v, got panic: %v", test.expectedPanic, r != nil)
				}
			}()

			formatFunc := getFormatter(test.formatName)
			if !test.expectedPanic {
				localProperty := test.property
				formatFunc(&localProperty, time.Time{})
				test.expectedFunc(&test.property, time.Time{})
				if localProperty.Value != test.property.Value || localProperty.Key != test.property.Key {
					t.Errorf("GetFormatter(%s) returned incorrect function", test.formatName)
				}
			} else {
				t.Errorf("GetFormatter(%s) did not panic as expected", test.formatName)
			}
		})
	}
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := parseName(tc.input)

			prop := &property{}

			f(prop, nil)

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
		tc := tc
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := parseMaxlen(tc.input)

			prop := &property{Value: tc.raw}

			f(prop, nil)

			if prop.Value != tc.expectedResult {
				t.Errorf("Expected value %s, but got %s", tc.expectedResult, prop.Value)
			}
		})
	}
}
