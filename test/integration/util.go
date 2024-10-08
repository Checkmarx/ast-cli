//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

var projectNameRandom = uuid.New().String()

const (
	ProjectNameFile = "projectName.txt"
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

func GenerateRandomProjectNameForScan() string {
	projectName := fmt.Sprintf("ast-cli-scan-%s", uuid.New().String())
	_ = WriteProjectNameToFile(projectName)
	return projectName
}

func WriteProjectNameToFile(projectName string) error {
	f, err := os.OpenFile(ProjectNameFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(projectName + "\n"); err != nil {
		return err
	}
	return nil
}

func getScsRepoToken() string {
	_ = viper.BindEnv("PERSONAL_ACCESS_TOKEN")
	return viper.GetString("PERSONAL_ACCESS_TOKEN")
}
