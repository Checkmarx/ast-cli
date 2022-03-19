package wrappers

type GitLabGroup struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	FullPath   string `json:"full_path"`
	Visibility string `json:"visibility"`
}

type GitLabProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNameSpace string `json:"path_with_namespace"`
	Visibility        string `json:"visibility"`
	DefaultBranch     string `json:"default_branch"`
}

type GitLabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Bot      bool   `json:"bot"`
}

type GitLabCommit struct {
	Name  string `json:"author_name"`
	Email string `json:"author_email"`
}

type GitLabWrapper interface {
	GetGitLabProjectsForUser() ([]GitLabProject, error)
	GetGitLabGroups(groupName string) ([]GitLabGroup, error)
	GetGitLabProjects(gitLabGroup GitLabGroup, queryParams map[string]string) ([]GitLabProject, error)
	GetCommits(gitLabProjectPathWithNameSpace string, queryParams map[string]string) ([]GitLabCommit, error)
}
