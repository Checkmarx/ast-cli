package wrappers

type rootApi struct {
	RepositoryURL   string `json:"repository_url"`
	OrganizationURL string `json:"organization_url"`
}

type Organization struct {
	RepositoriesURL string `json:"repos_url"`
}

type Repository struct {
	FullName   string `json:"full_name"`
	CommitsURL string `json:"commits_url"`
}

type Commit struct {
	CommitData CommitData `json:"commit"`
}

type CommitData struct {
	Author Author `json:"author"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitHubWrapper interface {
	GetOrganization(organizationName string) (Organization, error)
	GetRepository(organizationName, repositoryName string) (Repository, error)
	GetRepositories(organization Organization) ([]Repository, error)
	GetCommits(repository Repository, params map[string]string) ([]Commit, error)
}
