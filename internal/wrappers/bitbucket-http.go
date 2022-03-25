package wrappers

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

type BitBucketHTTPWrapper struct {
	client *http.Client
}

const (
	bitBucketBaseWorkspaceURL = "%sworkspaces/%s"
	bitBucketBaseRepoURL      = "%srepositories/%s"
	bitBucketBaseRepoNameURL  = "%srepositories/%s/%s"
	bitBucketBaseCommitURL    = "%srepositories/%s/%s/commits"
	failedBitbucketNotFound   = "no workspace with the provided identifier"
	failedBitbucketAuth       = "failed Bitbucket Authentication"
	pageLen                   = "pagelen"
	pageLenValue              = "100"
	page                      = "page"
	commitType                = "commit"
	repoType                  = "repo"
)

func NewBitbucketWrapper() BitBucketWrapper {
	return &BitBucketHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *BitBucketHTTPWrapper) GetworkspaceUUID(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (BitBucketRootWorkspace, error) {
	var err error
	var workspace BitBucketRootWorkspace
	var queryParams = make(map[string]string)

	workspaceURL := fmt.Sprintf(bitBucketBaseWorkspaceURL, bitBucketURL, workspaceName)

	err = g.get(workspaceURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &workspace, queryParams)

	return workspace, err
}

func (g *BitBucketHTTPWrapper) GetRepoUUID(bitBucketURL, workspaceName, repoName, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepo, error) {
	var err error
	var repo BitBucketRootRepo
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseRepoNameURL, bitBucketURL, workspaceName, repoName)
	err = g.get(repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &repo, queryParams)

	return repo, err
}

func (g *BitBucketHTTPWrapper) GetCommits(bitBucketURL, workspaceUUID, repoUUID, bitBucketUsername, bitBucketPassword string) (BitBucketRootCommit, error) {
	var err error
	var commits BitBucketRootCommit
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseCommitURL, bitBucketURL, workspaceUUID, repoUUID)
	pages, err := getWithPaginationBitBucket(g.client, repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), queryParams)
	if err != nil {
		return commits, err
	}
	// Go throw each commits in different pages
	for _, page := range pages {
		marshal, err := json.Marshal(page)
		if err != nil {
			return commits, err
		}
		commitHolder := BitBucketRootCommit{}
		json.Unmarshal(marshal, &commitHolder)
		for _, pageCommit := range commitHolder.Commits {
			// Filter the commits older than three months from the commits list
			if verifyDate(pageCommit) == false {
				return commits, nil
			}
			// Append the commit to the returned commits list
			commits.Commits = append(commits.Commits, pageCommit)
		}
	}

	return commits, err
}

func (g *BitBucketHTTPWrapper) GetRepositories(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepoList, error) {
	var err error
	var repos BitBucketRootRepoList
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseRepoURL, bitBucketURL, workspaceName)
	pages, err := getWithPaginationBitBucket(g.client, repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), queryParams)
	if err != nil {
		return repos, err
	}
	// Goes throw each repo in different pages
	for _, page := range pages {
		marshal, err := json.Marshal(page)
		if err != nil {
			return repos, err
		}
		repoHolder := BitBucketRootRepoList{}
		json.Unmarshal(marshal, &repoHolder)
		for _, pageCommit := range repoHolder.Values {
			// Append the commit to the returned commits list
			repos.Values = append(repos.Values, pageCommit)
		}
	}
	return repos, err
}

func (g *BitBucketHTTPWrapper) get(url, token string, target interface{}, queryParams map[string]string) error {
	var err error

	PrintIfVerbose(fmt.Sprintf("Request to %s", url))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(basicFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return err
		}
		// State sent when expired token
	case http.StatusUnauthorized:
		err = errors.New(failedBitbucketAuth)
		return err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New(failedBitbucketAuth)
		return err
	case http.StatusNotFound:
		err = errors.New(failedBitbucketNotFound)
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

func encodeBitBucketAuth(username, password string) string {
	return b64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func verifyDate(commit BitBucketCommit) bool {
	// Get last three months in string and cast it to date
	threeMonthsString := getThreeMonthsTime()
	threeMonths, _ := time.Parse(azureLayoutTime, threeMonthsString)

	commitDate, _ := time.Parse(time.RFC3339, commit.Date)
	// Check if the commit date occurs after the last three months
	if !commitDate.After(threeMonths) {
		return false
	}
	return true
}

func getWithPaginationBitBucket(
	client *http.Client,
	url string,
	token string,
	queryParams map[string]string,
) ([]interface{}, error) {

	var pageCollection = make([]interface{}, 0)

	var currentPage = 1
	var err error
	for currentPage != -1 {
		currentPage, err = collectPageBitBucket(client, token, url, currentPage, queryParams, &pageCollection, commitType)
		if err != nil {
			return nil, err
		}
	}
	return pageCollection, nil
}

func collectPageBitBucket(
	client *http.Client,
	token string,
	url string,
	currentPage int,
	queryParams map[string]string,
	pageCollection *[]interface{},
	types string,
) (int, error) {
	var holder BitBucketPage

	// Set the api page number
	queryParams[page] = strconv.Itoa(currentPage)
	// Set the api page length
	queryParams[pageLen] = pageLenValue
	_, err := getBitBucket(client, token, url, &holder, queryParams)
	if err != nil {
		return -1, err
	}

	// Verify if there is a next page
	if holder.Next != "" {
		currentPage += 1
	} else {
		currentPage = -1
	}

	*pageCollection = append(*pageCollection, holder)
	// In case the request is to get commits, verify if we should make next request based on the last commit date
	if types == commitType {
		marshal, err := json.Marshal(holder)
		if err != nil {
			return -1, err
		}
		holder1 := BitBucketRootCommit{}
		json.Unmarshal(marshal, &holder1)
		if !verifyDate(holder1.Commits[len(holder1.Commits)-1]) {
			return -1, nil
		}
	}

	return currentPage, nil
}

func getBitBucket(client *http.Client, token, url string, target interface{}, queryParams map[string]string) (*http.Response, error) {
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(basicFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}

	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	PrintIfVerbose(fmt.Sprintf("Request to %s", req.URL.String()))

	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return nil, err
		}
		// State sent when expired token
	case http.StatusUnauthorized:
		err = errors.New(failedBitbucketAuth)
		return nil, err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New(failedBitbucketAuth)
		return nil, err
	case http.StatusNotFound:
		err = errors.New(failedBitbucketNotFound)
		return nil, err
		// Case the commit/project does not exist in the organization
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(body))
	}
	return nil, nil
}
