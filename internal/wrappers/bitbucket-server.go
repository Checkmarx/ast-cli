package wrappers

type BitBucketServerCommitList struct {
	Commits       []BitBucketServerCommit `json:"values,omitempty"`
	IsLastPage    bool                    `json:"isLastPage"`
	Start         uint64                  `json:"start"`
	NextPageStart uint64                  `json:"nextPageStart"`
	Limit         uint64                  `json:"limit"`
	Size          uint64                  `json:"size"`
}

type BitBucketServerCommit struct {
	Author          BitBucketServerAuthor `json:"author,omitempty"`
	AuthorTimestamp uint64                `json:"authorTimestamp"`
}

type BitBucketServerAuthor struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type BitBucketServerRepo struct {
	Id   uint64 `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type BitBucketServerRepoList struct {
	Repos         []BitBucketServerRepo `json:"values"`
	IsLastPage    bool                  `json:"isLastPage"`
	Start         uint64                `json:"start"`
	NextPageStart uint64                `json:"nextPageStart"`
	Limit         uint64                `json:"limit"`
	Size          uint64                `json:"size"`
}

type BitBucketServerProject struct {
	Key  string `json:"key"`
	Id   uint64 `json:"id"`
	Name string `json:"name"`
}

type BitBucketServerProjectList struct {
	Projects      []BitBucketServerProject `json:"values"`
	IsLastPage    bool                     `json:"isLastPage"`
	Start         uint64                   `json:"start"`
	NextPageStart uint64                   `json:"nextPageStart"`
	Limit         uint64                   `json:"limit"`
	Size          uint64                   `json:"size"`
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
