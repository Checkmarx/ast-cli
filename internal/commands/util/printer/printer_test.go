package printer

import (
	"fmt"
	"os"
	"testing"

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
