package bitbucket_server

type BitBucketServerCommitList struct {
	Commits       []BitBucketServerCommit `json:"values,omitempty"`
	IsLastPage    bool                    `json:"isLastPage"`
	Start         int                     `json:"start"`
	NextPageStart int                     `json:"nextPageStart"`
	Limit         int                     `json:"limit"`
	Size          int                     `json:"size"`
}

type BitBucketServerCommit struct {
	Author          BitBucketServerAuthor `json:"author,omitempty"`
	AuthorTimestamp int                   `json:"authorTimestamp"`
}

type BitBucketServerAuthor struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type BitBucketServerRepo struct {
	Id   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type BitBucketServerRepoList struct {
	Repos         []BitBucketServerRepo `json:"values"`
	IsLastPage    bool                  `json:"isLastPage"`
	Start         int                   `json:"start"`
	NextPageStart int                   `json:"nextPageStart"`
	Limit         int                   `json:"limit"`
	Size          int                   `json:"size"`
}

type BitBucketServerProject struct {
	Key  string `json:"key"`
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type BitBucketServerProjectList struct {
	Projects      []BitBucketServerProject `json:"values"`
	IsLastPage    bool                     `json:"isLastPage"`
	Start         int                      `json:"start"`
	NextPageStart int                      `json:"nextPageStart"`
	Limit         int                      `json:"limit"`
	Size          int                      `json:"size"`
}

type BitBucketServerWrapper interface {
	GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) (
		[]BitBucketServerCommit,
		error,
	)
	GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) (
		[]BitBucketServerRepo,
		error,
	)
	GetProjects(bitBucketURL, bitBucketPassword string) (
		[]string,
		error,
	)
}
