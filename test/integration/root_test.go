// +build integration

package integration

import (
	"log"
	"os"
	"testing"
)

const (
	FullScanWait  = 60
	IncScanWait   = 30
	ScanPollSleep = 5
)

var Tags = map[string]string{
	"it_test_tag_1": "",
	"it_test_tag_2": "val",
	"it_test_tag_3": "",
}

var Groups = []string{
	"it_test_group_1",
	"it_test_group_2",
}

func TestMain(m *testing.M) {
	log.Println("CLI integration tests started")
	exitVal := m.Run()
	log.Println("CLI integration tests done")
	os.Exit(exitVal)
}
