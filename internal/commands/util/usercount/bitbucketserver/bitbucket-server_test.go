package bitbucketserver

import (
	"bytes"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
	"log"
	"testing"
)

func TestSearchReposWithCorruptedRepositories(t *testing.T) {

	mockWrapper := mock.WrapperBitbucketServer{
		CorruptedRepos: []string{"repo-2"},
	}

	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)
	defer log.SetOutput(nil)

	project := "mock-project"
	repos := []string{"repo-1", "repo-2", "repo-3"}
	token := "mock-token"

	views, viewsUsers, err := mockWrapper.SearchRepos(project, repos, token)

	assert.NilError(t, err, "SearchRepos should not return an error")

	assert.Equal(t, len(views), 2, "Only valid repositories should be processed")
	assert.Equal(t, views[0].Name, "mock-project/repo-1", "First repository name should match")
	assert.Equal(t, views[1].Name, "mock-project/repo-3", "Second repository name should match")

	assert.Equal(t, len(viewsUsers), 2, "Each repository should have 1 contributor")
	assert.Equal(t, viewsUsers[0].Name, "mock-project/repo-1", "Contributor should match first repository")
	assert.Equal(t, viewsUsers[1].Name, "mock-project/repo-3", "Contributor should match second repository")

	logStr := logOutput.String()
	assert.Assert(t, containsLog(logStr, "Skipping repository mock-project/repo-2: Repository is corrupted"), "Log should contain corrupted repository message")
	assert.Assert(t, containsLog(logStr, "Processed repository mock-project/repo-1"), "Log should confirm successful processing of repo-1")
	assert.Assert(t, containsLog(logStr, "Processed repository mock-project/repo-3"), "Log should confirm successful processing of repo-3")

	t.Log("Captured Logs:")
	t.Log(logStr)
}

func containsLog(logStr, expected string) bool {
	return bytes.Contains([]byte(logStr), []byte(expected))
}
