package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type GitLabMockWrapper struct {
}

func (g GitLabMockWrapper) GetGitLabProjectsForUser() ([]wrappers.GitLabProject, error) {
	return []wrappers.GitLabProject{}, nil
}

func (g GitLabMockWrapper) GetGitLabProjects(
	gitLabGroupName string, queryParams map[string]string,
) ([]wrappers.GitLabProject, error) {
	return []wrappers.GitLabProject{}, nil
}

func (g GitLabMockWrapper) GetCommits(
	gitLabProjectPathWithNameSpace string, queryParams map[string]string,
) ([]wrappers.GitLabCommit, error) {
	return []wrappers.GitLabCommit{}, nil
}
