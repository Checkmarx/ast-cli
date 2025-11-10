package bitbucketserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/viper"
)

type HTTPWrapper struct {
	client *http.Client
}

type ErrorModel struct {
	Context string `json:"context,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorResponseModel struct {
	Errors []ErrorModel `json:"errors,omitempty"`
}

const (
	bitBucketServerProjectsURL  = "rest/api/1.0/projects/"
	bitBucketServerReposURL     = "rest/api/1.0/projects/%s/repos/"
	bitBucketServerCommitsURL   = "rest/api/1.0/projects/%s/repos/%s/commits"
	bitBucketPageLimit          = 100
	bitBucketServerPageStart    = "start"
	bitBucketServerPageLimit    = "limit"
	bitBucketServerAuthError    = "failed Bitbucket Server authentication"
	bitBucketServerNotFound     = "%s, skipping: %s"
	bitBucketServerBearerFormat = "Bearer %s"
)

var (
	ErrNotFound    = errors.New("resource not found")
	ninetyDaysPast = time.Now().AddDate(0, -3, 0)
)

func NewBitbucketServerWrapper() Wrapper {
	return &HTTPWrapper{
		client: wrappers.GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (b HTTPWrapper) GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) (
	[]Commit,
	error,
) {
	url := bitBucketURL + fmt.Sprintf(bitBucketServerCommitsURL, projectKey, repoSlug)

	var acc []Commit

	pageHolder := CommitList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	for !pageHolder.IsLastPage {
		queryParams := buildQueryParams(pageHolder.NextPageStart)

		err := getBitBucketServer(b.client, bitBucketPassword, url, &pageHolder, queryParams)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				break
			}
			return nil, err
		}

		for _, commit := range pageHolder.Commits {
			timestamp := time.UnixMilli(commit.AuthorTimestamp)
			if timestamp.After(ninetyDaysPast) {
				acc = append(acc, commit)
			} else {
				break
			}
		}
	}

	return acc, nil
}

func (b HTTPWrapper) GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) (
	[]Repo,
	error,
) {
	url := bitBucketURL + fmt.Sprintf(bitBucketServerReposURL, projectKey)

	var acc []Repo

	pageHolder := RepoList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	for !pageHolder.IsLastPage {
		queryParams := buildQueryParams(pageHolder.NextPageStart)

		err := getBitBucketServer(b.client, bitBucketPassword, url, &pageHolder, queryParams)
		if err != nil {
			return nil, err
		}

		acc = append(acc, pageHolder.Repos...)
	}
	return acc, nil
}

func (b HTTPWrapper) GetProjects(bitBucketURL, bitBucketPassword string) (
	[]string,
	error,
) {
	url := bitBucketURL + bitBucketServerProjectsURL

	var acc []Project

	pageHolder := ProjectList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	for !pageHolder.IsLastPage {
		queryParams := buildQueryParams(pageHolder.NextPageStart)

		err := getBitBucketServer(b.client, bitBucketPassword, url, &pageHolder, queryParams)
		if err != nil {
			return nil, err
		}

		acc = append(acc, pageHolder.Projects...)
	}
	var projectKeys []string

	for _, project := range acc {
		projectKeys = append(projectKeys, project.Key)
	}

	return projectKeys, nil
}

func getBitBucketServer(
	client *http.Client,
	token, url string,
	target interface{},
	queryParams map[string]string,
) error {
	var err error

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	if len(token) > 0 {
		req.Header.Add(wrappers.AuthorizationHeader, fmt.Sprintf(bitBucketServerBearerFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}

	req.URL.RawQuery = q.Encode()
	resp, err := wrappers.WithSCMRateLimitRetry(
		wrappers.BitbucketRateLimitConfig,
		func() (*http.Response, error) {
			return client.Do(req)
		},
	)
	if err != nil {
		return err
	}
	logger.PrintRequest(req)

	defer func() {
		_ = resp.Body.Close()
	}()

	logger.PrintResponse(resp, true)

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return err
		}
	case http.StatusUnauthorized:
		err = errors.New(bitBucketServerAuthError)
		return err
	case http.StatusForbidden:
		err = errors.New(bitBucketServerAuthError)
		return err
	case http.StatusNotFound:
		errorResponse := ErrorResponseModel{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}
		for _, errorModel := range errorResponse.Errors {
			logger.Printf(bitBucketServerNotFound, errorModel.Message, url)
		}
		return ErrNotFound
		// Case the commit/project does not exist in the organization
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func buildQueryParams(nextPageStart int) map[string]string {
	queryParams := map[string]string{}
	queryParams[bitBucketServerPageStart] = strconv.Itoa(nextPageStart)
	queryParams[bitBucketServerPageLimit] = strconv.Itoa(bitBucketPageLimit)
	return queryParams
}
