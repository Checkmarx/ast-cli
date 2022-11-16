package bitbucketserver

type CommitList struct {
	Commits       []Commit `json:"values,omitempty"`
	IsLastPage    bool     `json:"isLastPage"`
	Start         int      `json:"start"`
	NextPageStart int      `json:"nextPageStart"`
	Limit         int      `json:"limit"`
	Size          int      `json:"size"`
}

type Commit struct {
	Author          Author `json:"author,omitempty"`
	AuthorTimestamp int    `json:"authorTimestamp"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type Repo struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type RepoList struct {
	Repos         []Repo `json:"values"`
	IsLastPage    bool   `json:"isLastPage"`
	Start         int    `json:"start"`
	NextPageStart int    `json:"nextPageStart"`
	Limit         int    `json:"limit"`
	Size          int    `json:"size"`
}

type Project struct {
	Key  string `json:"key"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProjectList struct {
	Projects      []Project `json:"values"`
	IsLastPage    bool      `json:"isLastPage"`
	Start         int       `json:"start"`
	NextPageStart int       `json:"nextPageStart"`
	Limit         int       `json:"limit"`
	Size          int       `json:"size"`
}

type Wrapper interface {
	GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) (
		[]Commit,
		error,
	)
	GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) (
		[]Repo,
		error,
	)
	GetProjects(bitBucketURL, bitBucketPassword string) (
		[]string,
		error,
	)
}
