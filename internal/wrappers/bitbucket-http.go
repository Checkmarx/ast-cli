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

	"github.com/checkmarx/ast-cli/internal/logger"
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
		client: GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *BitBucketHTTPWrapper) GetworkspaceUUID(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (
	BitBucketRootWorkspace, error,
) {
	var err error
	var workspace BitBucketRootWorkspace
	var queryParams = make(map[string]string)

	workspaceURL := fmt.Sprintf(bitBucketBaseWorkspaceURL, bitBucketURL, workspaceName)

	err = g.getFromBitBucket(
		workspaceURL,
		encodeBitBucketAuth(bitBucketUsername, bitBucketPassword),
		&workspace,
		queryParams,
	)

	return workspace, err
}

func (g *BitBucketHTTPWrapper) GetRepoUUID(bitBucketURL, workspaceName, repoName, bitBucketUsername, bitBucketPassword string) (
	BitBucketRootRepo, error,
) {
	var err error
	var repo BitBucketRootRepo
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseRepoNameURL, bitBucketURL, workspaceName, repoName)
	err = g.getFromBitBucket(repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &repo, queryParams)

	return repo, err
}

func (g *BitBucketHTTPWrapper) GetCommits(bitBucketURL, workspaceUUID, repoUUID, bitBucketUsername, bitBucketPassword string) (
	BitBucketRootCommit, error,
) {
	var commits BitBucketRootCommit
	var queryParams = make(map[string]string)
	repoURL := fmt.Sprintf(bitBucketBaseCommitURL, bitBucketURL, workspaceUUID, repoUUID)
	pages, err := getWithPaginationBitBucket(
		g.client,
		repoURL,
		encodeBitBucketAuth(bitBucketUsername, bitBucketPassword),
		commitType,
		queryParams,
	)
	if err != nil {
		return commits, err
	}
	// Goes throw each commits in different pages
	for _, page := range pages {
		marshal, errM := json.Marshal(page)
		if errM != nil {
			return commits, errM
		}
		commitHolder := BitBucketRootCommit{}
		err = json.Unmarshal(marshal, &commitHolder)
		if err != nil {
			return commits, err
		}
		for _, pageCommit := range commitHolder.Commits {
			// Filter the commits older than three months from the commits list
			if !verifyDate(pageCommit) {
				return commits, nil
			}
			// Append the commit to the returned commits list
			commits.Commits = append(commits.Commits, pageCommit)
		}
	}

	return commits, err
}

func (g *BitBucketHTTPWrapper) GetRepositories(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (
	BitBucketRootRepoList, error,
) {
	var repos BitBucketRootRepoList
	var queryParams = make(map[string]string)
	repoURL := fmt.Sprintf(bitBucketBaseRepoURL, bitBucketURL, workspaceName)
	pages, err := getWithPaginationBitBucket(
		g.client,
		repoURL,
		encodeBitBucketAuth(bitBucketUsername, bitBucketPassword),
		repoType,
		queryParams,
	)
	if err != nil {
		return repos, err
	}
	// Goes throw each repo in different pages
	for _, page := range pages {
		marshal, errM := json.Marshal(page)
		if errM != nil {
			return repos, errM
		}
		repoHolder := BitBucketRootRepoList{}
		err = json.Unmarshal(marshal, &repoHolder)
		if err != nil {
			return repos, err
		}
		repos.Values = append(repos.Values, repoHolder.Values...)
	}
	return repos, err
}

func (g *BitBucketHTTPWrapper) getFromBitBucket(
	url, token string, target interface{}, queryParams map[string]string,
) error {
	var err error

	logger.PrintIfVerbose(fmt.Sprintf("Request to %s", url))

	resp, err := GetWithQueryParams(g.client, url, token, basicFormat, queryParams)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
	return commitDate.After(threeMonths)
}

func getWithPaginationBitBucket(
	client *http.Client,
	url,
	token,
	types string,
	queryParams map[string]string,
) ([]interface{}, error) {
	var pageCollection = make([]interface{}, 0)
	var currentPage = 1
	var err error
	for currentPage != -1 {
		currentPage, err = collectPageBitBucket(client, token, url, types, currentPage, queryParams, &pageCollection)
		if err != nil {
			return nil, err
		}
	}
	return pageCollection, nil
}

func collectPageBitBucket(
	client *http.Client,
	token,
	url,
	types string,
	currentPage int,
	queryParams map[string]string,
	pageCollection *[]interface{},
) (int, error) {
	var holder BitBucketPage

	// Set the api page number
	queryParams[page] = strconv.Itoa(currentPage)
	// Set the api page length
	queryParams[pageLen] = pageLenValue
	err := getBitBucket(client, token, url, &holder, queryParams)
	if err != nil {
		return -1, err
	}

	// Verify if there is a next page
	if holder.Next != "" {
		currentPage++
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
		commitHolder := BitBucketRootCommit{}
		err = json.Unmarshal(marshal, &commitHolder)
		if err != nil || len(commitHolder.Commits) == 0 || !verifyDate(commitHolder.Commits[len(commitHolder.Commits)-1]) {
			return -1, err
		}
	}
	return currentPage, nil
}

func getBitBucket(client *http.Client, token, url string, target interface{}, queryParams map[string]string) error {
	resp, err := GetWithQueryParams(client, url, token, basicFormat, queryParams)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
