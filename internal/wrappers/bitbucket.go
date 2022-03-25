package wrappers

type BitBucketRootCommit struct {
	Commits []BitBucketCommit `json:"values,omitempty"`
	Next    string            `json:"next"`
}

type BitBucketCommit struct {
	Author BitBucketAuthor `json:"author,omitempty"`
	Date   string          `json:"date"`
}

type BitBucketAuthor struct {
	Name string `json:"raw"`
}

type BitBucketRootWorkspace struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type BitBucketRootRepo struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type BitBucketRootRepoList struct {
	Values []BitBucketRepo `json:"values"`
	Next   string          `json:"next"`
}

type BitBucketRepo struct {
	Name string `json:"full_name"`
	UUID string `json:"uuid"`
}

type BitBucketPage struct {
	Next   string      `json:"next"`
	Values interface{} `json:"values,omitempty"`
}

type BitBucketWrapper interface {
	GetworkspaceUUID(bitBucketURL, workspace, bitBucketUsername, bitBucketPassword string) (BitBucketRootWorkspace, error)
	GetRepoUUID(bitBucketURL, workspaceName, repo, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepo, error)
	GetCommits(bitBucketURL, workspaceUUID, repoUUID, bitBucketUsername, bitBucketPassword string) (BitBucketRootCommit, error)
	GetRepositories(bitBucketURL, workspaceUUID, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepoList, error)
}
