package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type GitHubMockWrapper struct {
}

func (g GitHubMockWrapper) GetOrganization(string) (wrappers.Organization, error) {
	return wrappers.Organization{}, nil
}

func (g GitHubMockWrapper) GetRepository(string, string) (wrappers.Repository, error) {
	return wrappers.Repository{}, nil
}

func (g GitHubMockWrapper) GetRepositories(wrappers.Organization) ([]wrappers.Repository, error) {
	return []wrappers.Repository{{}}, nil
}

func (g GitHubMockWrapper) GetCommits(wrappers.Repository, map[string]string) ([]wrappers.CommitRoot, error) {
	return []wrappers.CommitRoot{}, nil
}
