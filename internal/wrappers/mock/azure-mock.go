package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type AzureMockWrapper struct {
}

func (g AzureMockWrapper) GetProjects(string, string, string) (wrappers.AzureRootProject, error) {
	return wrappers.AzureRootProject{}, nil
}

func (g AzureMockWrapper) GetCommits(string, string, string, string, string) (wrappers.AzureRootCommit, error) {
	return wrappers.AzureRootCommit{}, nil
}

func (g AzureMockWrapper) GetRepositories(string, string, string, string) (wrappers.AzureRootRepo, error) {
	return wrappers.AzureRootRepo{}, nil
}