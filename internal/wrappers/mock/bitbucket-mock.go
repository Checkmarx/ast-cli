package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type BitBucketMockWrapper struct {
}

func (g BitBucketMockWrapper) GetworkspaceUUID(bitBucketURL, workspace, bitBucketUsername, bitBucketPassword string) (wrappers.BitBucketRootWorkspace, error) {
	return wrappers.BitBucketRootWorkspace{UUID: "{MOCK UUID}", Name: "MOCK NAME"}, nil
}

func (g BitBucketMockWrapper) GetRepoUUID(bitBucketURL, workspaceName, repo, bitBucketUsername, bitBucketPassword string) (wrappers.BitBucketRootRepo, error) {
	return wrappers.BitBucketRootRepo{UUID: "{MOCK UUID}", Name: "MOCK NAME"}, nil
}

func (g BitBucketMockWrapper) GetCommits(bitBucketURL, workspaceUUID, repoUUID, bitBucketUsername, bitBucketPassword string) (wrappers.BitBucketRootCommit, error) {
	if len(workspaceUUID) > 0 {
		var commits = make([]wrappers.BitBucketCommit, 1)
		author := wrappers.BitBucketAuthor{Name: "MOCK NAME"}
		commits[0] = wrappers.BitBucketCommit{
			Author: author,
			Date:   "2021-12-16T10:25:28+00:00",
		}
		return wrappers.BitBucketRootCommit{Commits: commits}, nil
	}
	return wrappers.BitBucketRootCommit{}, nil
}

func (g BitBucketMockWrapper) GetRepositories(bitBucketURL, workspaceUUID, bitBucketUsername, bitBucketPassword string) (wrappers.BitBucketRootRepoList, error) {
	if len(workspaceUUID) > 0 {
		var repos = make([]wrappers.BitBucketRepo, 1)
		repos[0] = wrappers.BitBucketRepo{
			Name: "MOCK REPO",
			UUID: "{MOCK UUID}",
		}
		return wrappers.BitBucketRootRepoList{Values: repos}, nil
	}
	return wrappers.BitBucketRootRepoList{}, nil
}
