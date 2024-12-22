package mock

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"log"
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

type SimulatedWrapper struct {
}
type repositoryView struct {
	Name               string `json:"name"`
	UniqueContributors uint64 `json:"unique_contributors"`
}

type userView struct {
	Name                       string `json:"name"`
	UniqueContributorsUsername string `json:"unique_contributors_username"`
}

func (g SimulatedWrapper) GetRepositories(bitBucketURL, project, bitBucketToken string) (wrappers.BitBucketRootRepoList, error) {
	return wrappers.BitBucketRootRepoList{
		Values: []wrappers.BitBucketRepo{
			{Name: "repo-1", UUID: "repo-1"},
			{Name: "repo-2", UUID: "repo-2"},
			{Name: "repo-3", UUID: "repo-3"},
		},
	}, nil
}

func (g SimulatedWrapper) GetCommits(bitBucketURL, project, repoUUID, bitBucketToken string) ([]wrappers.BitBucketCommit, error) {
	if repoUUID == "repo-2" {
		return nil, errors.New(fmt.Sprintf("repository %s is corrupted", repoUUID))
	}
	return []wrappers.BitBucketCommit{
		{
			Author: wrappers.BitBucketAuthor{Name: "Mock Author"},
			Date:   "2021-12-16T10:25:28+00:00",
		},
	}, nil
}

func (g SimulatedWrapper) SearchRepos(
	project string,
	repos []string,
	bitBucketToken string,
) ([]repositoryView, []userView, error) {
	var views []repositoryView
	var viewsUsers []userView
	var totalCommits []wrappers.BitBucketCommit

	for _, repo := range repos {
		commits, err := g.GetCommits("mock-url", project, repo, bitBucketToken)
		if err != nil {
			log.Printf("Skipping repository %s/%s: Repository is corrupted (error: %v)", project, repo, err)
			continue
		}

		log.Printf("Processed repository %s/%s", project, repo)

		totalCommits = append(totalCommits, commits...)

		uniqueContributors := map[string]string{
			"mock-email": "Mock Author",
		}
		views = append(
			views,
			repositoryView{
				Name:               fmt.Sprintf("%s/%s", project, repo),
				UniqueContributors: uint64(len(uniqueContributors)),
			},
		)
		for email, name := range uniqueContributors {
			viewsUsers = append(
				viewsUsers,
				userView{
					Name:                       fmt.Sprintf("%s/%s", project, repo),
					UniqueContributorsUsername: fmt.Sprintf("%s - %s", name, email),
				},
			)
		}
	}
	return views, viewsUsers, nil
}
