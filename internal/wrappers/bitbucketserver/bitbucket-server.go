package bitbucketserver

type BitbucketServerCommitList struct {
	Commits       []BitbucketServerCommit `json:"values,omitempty"`
	IsLastPage    bool                    `json:"isLastPage"`
	Start         int                     `json:"start"`
	NextPageStart int                     `json:"nextPageStart"`
	Limit         int                     `json:"limit"`
	Size          int                     `json:"size"`
}

type BitbucketServerCommit struct {
	Author          BitbucketServerAuthor `json:"author,omitempty"`
	AuthorTimestamp int64                 `json:"authorTimestamp"`
}

type BitbucketServerAuthor struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type BitbucketServerRepo struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type BitbucketServerRepoList struct {
	Repos         []BitbucketServerRepo `json:"values"`
	IsLastPage    bool                  `json:"isLastPage"`
	Start         int                   `json:"start"`
	NextPageStart int                   `json:"nextPageStart"`
	Limit         int                   `json:"limit"`
	Size          int                   `json:"size"`
}

type BitbucketServerProject struct {
	Key  string `json:"key"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type BitbucketServerProjectList struct {
	Projects      []BitbucketServerProject `json:"values"`
	IsLastPage    bool                     `json:"isLastPage"`
	Start         int                      `json:"start"`
	NextPageStart int                      `json:"nextPageStart"`
	Limit         int                      `json:"limit"`
	Size          int                      `json:"size"`
}

type BitbucketServerWrapper interface {
	GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) (
		[]BitbucketServerCommit,
		error,
	)
	GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) (
		[]BitbucketServerRepo,
		error,
	)
	GetProjects(bitBucketURL, bitBucketPassword string) (
		[]string,
		error,
	)
}
