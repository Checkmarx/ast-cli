package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type GitLabMockWrapper struct {
}

func (g GitLabMockWrapper) GetGitLabProjectsForUser() ([]wrappers.GitLabProject, error) {
	return []wrappers.GitLabProject{}, nil
}

func (g GitLabMockWrapper) GetGitLabGroups(groupName string) ([]wrappers.GitLabGroup, error) {
	return []wrappers.GitLabGroup{}, nil
}

func (g GitLabMockWrapper) GetGitLabProjects(gitLabGroup wrappers.GitLabGroup, queryParams map[string]string) ([]wrappers.GitLabProject, error) {
	return []wrappers.GitLabProject{}, nil
}

func (g GitLabMockWrapper) GetCommits(gitLabProjectId string, queryParams map[string]string) ([]wrappers.GitLabCommit, error) {
	return []wrappers.GitLabCommit{}, nil
}
