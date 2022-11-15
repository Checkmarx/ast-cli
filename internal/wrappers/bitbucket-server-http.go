package wrappers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

type BitBucketServerHTTPWrapper struct {
	client *http.Client
}

const (
	bitBucketServerApiPrefix = "rest/api/1.0/"
	bitBucketServerProjects  = "projects/%s/"
	bitBucketServerRepos     = "repos/%s"
)

func NewBitbucketServerWrapper() BitBucketServerWrapper {
	return &BitBucketServerHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (b BitBucketServerHTTPWrapper) GetCommits(bitBucketURL, projectKey, repoSlug, bitBucketPassword string) (
	[]BitBucketServerCommit,
	error,
) {
	url := bitBucketURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += bitBucketServerApiPrefix
	url += fmt.Sprintf(bitBucketServerProjects, projectKey)
	url += fmt.Sprintf(bitBucketServerRepos, repoSlug)
	url += "/commits"

	var acc []BitBucketServerCommit

	pageHolder := BitBucketServerCommitList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	logger.Print(url)
	for !pageHolder.IsLastPage {
		logger.Printf("Page start: %v", pageHolder.NextPageStart)
		queryParams := map[string]string{}
		queryParams["start"] = strconv.FormatUint(pageHolder.NextPageStart, 10)

		err := getBitBucketServer(b.client, bitBucketPassword, url, &pageHolder, queryParams)
		if err != nil {
			return nil, err
		}

		acc = append(acc, pageHolder.Commits...)
	}

	return acc, nil
}

func (b BitBucketServerHTTPWrapper) GetRepositories(bitBucketURL, projectKey, bitBucketPassword string) (
	[]BitBucketServerRepo,
	error,
) {
	url := bitBucketURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += bitBucketServerApiPrefix
	url += fmt.Sprintf(bitBucketServerProjects, projectKey)
	url += fmt.Sprintf(bitBucketServerRepos, "")

	var acc []BitBucketServerRepo

	pageHolder := BitBucketServerRepoList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	logger.Print(url)
	for !pageHolder.IsLastPage {
		pageHolder.IsLastPage = true
		logger.Printf("Page start: %v", pageHolder.NextPageStart)
		queryParams := map[string]string{}
		queryParams["start"] = strconv.FormatUint(pageHolder.NextPageStart, 10)

		err := getBitBucketServer(b.client, bitBucketPassword, url, &pageHolder, queryParams)
		if err != nil {
			return nil, err
		}

		acc = append(acc, pageHolder.Repos...)
	}
	return acc, nil
}

func (b BitBucketServerHTTPWrapper) GetProjects(bitBucketURL, bitBucketPassword string) (
	[]string,
	error,
) {
	url := bitBucketURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += bitBucketServerApiPrefix
	url += "projects/"

	var acc []BitBucketServerProject

	pageHolder := BitBucketServerProjectList{}
	pageHolder.IsLastPage = false
	pageHolder.NextPageStart = 0
	logger.Print(url)
	for !pageHolder.IsLastPage {
		logger.Printf("Page start: %v", pageHolder.NextPageStart)
		queryParams := map[string]string{}
		queryParams["start"] = strconv.FormatUint(pageHolder.NextPageStart, 10)

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

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf("Bearer %s", token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}

	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
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
		// State sent when expired token
	case http.StatusUnauthorized:
		err = errors.New("failed Bitbucket Authentication")
		return err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New("failed Bitbucket Authentication")
		return err
	case http.StatusNotFound:
		err = errors.New("no workspace with the provided identifier")
		return err
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
