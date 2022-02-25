package wrappers

type rootAPI struct {
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

type CommitRoot struct {
	Commit Commit  `json:"commit"`
	Author *Author `json:"author,omitempty"`
}

type Author struct {
	Type string `json:"type"`
}

type Commit struct {
	CommitAuthor CommitAuthor `json:"author"`
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitHubWrapper interface {
	GetOrganization(organizationName string) (Organization, error)
	GetRepository(organizationName, repositoryName string) (Repository, error)
	GetRepositories(organization Organization) ([]Repository, error)
	GetCommits(repository Repository, queryParams map[string]string) ([]CommitRoot, error)
}
