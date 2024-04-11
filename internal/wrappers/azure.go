package wrappers

type AzureRootCommit struct {
	Commits []AzureCommit `json:"value,omitempty"`
}

type AzureCommit struct {
	Author AzureAuthor `json:"author,omitempty"`
}

type AzureAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type AzureRootRepo struct {
	Repos []AzureRepo `json:"value,omitempty"`
}

func (a *AzureRootRepo) GetEnabledRepos() AzureRootRepo {
	var enabledRepos []AzureRepo
	for _, repo := range a.Repos {
		if !repo.IsDisabled {
			enabledRepos = append(enabledRepos, repo)
		}
	}
	return AzureRootRepo{Repos: enabledRepos}
}

type AzureRepo struct {
	Name       string `json:"name"`
	IsDisabled bool   `json:"isDisabled"`
}

type AzureRootProject struct {
	Count    int            `json:"count,omitempty"`
	Projects []AzureProject `json:"value,omitempty"`
}

type AzureProject struct {
	Name string `json:"name"`
}

type AzureWrapper interface {
	GetProjects(url string, organizationName string, token string) (AzureRootProject, error)
	GetCommits(url, organizationName, projectName, repositoryName, token string) (AzureRootCommit, error)
	GetRepositories(url string, organizationName string, projectName string, token string) (AzureRootRepo, error)
}
