//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"
)

var projectNameRandom = uuid.New().String()

func formatTags(tags map[string]string) string {
	var tagsStr string
	for key := range tags {
		val := tags[key]
		tagsStr += key
		if val != "" {
			tagsStr += ":" + val
		}
		tagsStr += ","
	}
	tagsStr = strings.TrimRight(tagsStr, ",")
	return tagsStr
}

func getAllTags(t *testing.T, baseCmd string) map[string][]string {
	tagsCommand, buffer := createRedirectedTestCommand(t)

	err := execute(tagsCommand, baseCmd, "tags")
	assert.NilError(t, err, "Getting tags should pass")

	// Read response from buffer
	tags := map[string][]string{}
	_ = unmarshall(t, buffer, &tags, "Reading tags JSON should pass")

	return tags
}

func flag(f string) string {
	return "--" + f
}

func getProjectNameForTest() string {
	return fmt.Sprintf("ast-cli-tests_%s", projectNameRandom)
}
