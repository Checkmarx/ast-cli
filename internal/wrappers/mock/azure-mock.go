package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type AzureMockWrapper struct {
}

func (g AzureMockWrapper) GetProjects(url, organizationName, token string) (wrappers.AzureRootProject, error) {
	if len(organizationName) > 0 {
		var projects = make([]wrappers.AzureProject, 1)
		projects[0] = wrappers.AzureProject{
			Name: "MOCK",
		}
		return wrappers.AzureRootProject{Count: 1, Projects: projects}, nil
	}
	return wrappers.AzureRootProject{}, nil
}

func (g AzureMockWrapper) GetCommits(url, organizationName, projectName, repositoryName, token string) (wrappers.AzureRootCommit, error) {
	if len(repositoryName) > 0 {
		var commits = make([]wrappers.AzureCommit, 1)
		author := wrappers.AzureAuthor{Name: "MOCK NAME", Email: "MOCK Email"}
		commits[0] = wrappers.AzureCommit{
			Author: author,
		}
		return wrappers.AzureRootCommit{Commits: commits}, nil
	}
	return wrappers.AzureRootCommit{}, nil
}

func (g AzureMockWrapper) GetRepositories(url, organizationName, projectName, token string) (wrappers.AzureRootRepo, error) {
	if len(projectName) > 0 {
		var repos = make([]wrappers.AzureRepo, 1)
		repos[0] = wrappers.AzureRepo{
			Name: "MOCK REPO",
		}
		return wrappers.AzureRootRepo{Repos: repos}, nil
	}
	return wrappers.AzureRootRepo{}, nil
}
