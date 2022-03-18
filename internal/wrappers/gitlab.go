package wrappers

type GitLabGroup struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	FullPath   string `json:"full_path"`
	Visibility string `json:"visibility"`
}

type GitLabProject struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PathWithNameSpace string `json:"path_with_namespace"`
	Visibility        string `json:"visibility"`
	DefaultBranch     string `json:"default_branch"`
}

type GitLabUser struct {
	ID string `json:"id"`
}

type GitLabRootCommit struct {
	Commits []GitLabCommit `json:"value,omitempty"`
}

type GitLabCommit struct {
	Name  string `json:"author_name,omitempty"`
	Email string `json:"author_email,omitempty"`
}

type GitLabWrapper interface {
	GetGitLabProjectsForUser() ([]GitLabProject, error)
	GetGitLabGroups(groupName string) ([]GitLabGroup, error)
	GetGitLabProjects(gitLabGroup GitLabGroup, queryParams map[string]string) ([]GitLabProject, error)
	GetCommits(gitLabProjectPathWithNameSpace string, queryParams map[string]string) (GitLabRootCommit, error)
}
