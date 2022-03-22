package wrappers

type BitBucketRootCommit struct {
	Commits []BitBucketCommit `json:"values,omitempty"`
}

type BitBucketCommit struct {
	Author BitBucketAuthor `json:"author,omitempty"`
	Date   string          `json:"date"`
}

type BitBucketAuthor struct {
	Name string `json:"raw"`
}

type BitBucketRootWorkspace struct {
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

type BitBucketRootRepo struct {
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

type BitBucketRootRepoList struct {
	Values []BitBucketRepo `json:"values"`
}

type BitBucketRepo struct {
	Name string `json:"full_name"`
	Uuid string `json:"uuid"`
}

type BitBucketWrapper interface {
	GetWorkspaceUuid(bitBucketURL, workspace, bitBucketUsername, bitBucketPassword string) (BitBucketRootWorkspace, error)
	GetRepoUuid(bitBucketURL, workspaceName, repo, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepo, error)
	GetCommits(bitBucketURL, workspaceUuid, repoUuid, bitBucketUsername, bitBucketPassword string) (BitBucketRootCommit, error)
	GetRepositories(bitBucketURL, workspaceUuid, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepoList, error)
}
