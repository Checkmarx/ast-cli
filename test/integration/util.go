//go:build integration

package integration

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

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

func formatGroups(groups []string) string {
	var groupsStr string
	for _, group := range groups {
		groupsStr += group
		groupsStr += ","
	}
	groupsStr = strings.TrimRight(groupsStr, ",")
	return groupsStr
}

func contains(array []string, val string) bool {
	for _, e := range array {
		if e == val {
			return true
		}
	}
	return false
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
