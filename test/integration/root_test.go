// +build integration

package integration

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	exitVal := m.Run()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}
