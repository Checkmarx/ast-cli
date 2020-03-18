package commands

import (
	"log"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestMain(m *testing.M) {
	log.Println("Commands tests started")
	// Run all tests
	exitVal := m.Run()
	log.Println("Commands tests done")
	os.Exit(exitVal)
}

func TestNewAstCLI(t *testing.T) {
	cli := NewAstCLI("url", "scans", "projects", "uploads", "results")
	assert.Assert(t, cli != nil)
}
