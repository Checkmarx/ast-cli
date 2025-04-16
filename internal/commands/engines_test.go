package commands

import (
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
	"testing"
)

func TestNewEnginesCommand(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "")
	assert.NilError(t, err)
}

func TestNewEnginesCommandHelp(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "help")
	assert.NilError(t, err)
}

func TestNewEnginesSubCommand(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api")
	assert.NilError(t, err)
}

func TestNewEnginesSubCommandHelp(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--help")
	assert.NilError(t, err)
}

func TestSubCommandEngineType1(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--engine-name", "SAST")
	assert.NilError(t, err)
}

func TestSubCommandEngineType2(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--engine-name", "SCA")
	assert.NilError(t, err)
}

func TestSubCommandEngineType3(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--engine-name", "Iac")
	assert.NilError(t, err)
}

func TestSubCommandOutPutFormat1(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--output-format", "json")
	assert.NilError(t, err)
}

func TestSubCommandOutPutFormat2(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--output-format", "yaml")
	assert.NilError(t, err)
}

func TestSubCommandOutPutFormat3(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--output-format", "table")
	assert.NilError(t, err)
}

func TestSubCommandEngineError1(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--chibute", "SAST")
	assert.Assert(t, err != nil)
}

func TestSubCommandEngineError2(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--engine-name", "SASTS")
	assert.NilError(t, err)
}

func TestSubCommandEngineError3(t *testing.T) {
	cmd := NewEnginesCommand(&mock.NewHTTPEnginesMockWrapper{})
	err := executeTestCommand(cmd, "list-api", "--output-format", "jsonsa")
	assert.Assert(t, err != nil)
}
