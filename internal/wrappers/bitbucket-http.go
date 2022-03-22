package wrappers

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	failedBitbucketNotFound   = "No workspace with the provided identifier"
	failedBitbucketAuth       = "Failed Bitbucket Authentication"
)

func NewBitbucketWrapper() BitBucketWrapper {
	return &BitBucketHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *BitBucketHTTPWrapper) GetWorkspaceUuid(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (BitBucketRootWorkspace, error) {
	var err error
	var workspace BitBucketRootWorkspace
	var queryParams = make(map[string]string)

	workspaceURL := fmt.Sprintf(bitBucketBaseWorkspaceURL, bitBucketURL, workspaceName)

	err = g.get(workspaceURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &workspace, queryParams, basicFormat)

	return workspace, err
}

func (g *BitBucketHTTPWrapper) GetRepoUuid(bitBucketURL, workspaceName, repoName, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepo, error) {
	var err error
	var repo BitBucketRootRepo
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseRepoNameURL, bitBucketURL, workspaceName, repoName)
	err = g.get(repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &repo, queryParams, basicFormat)

	return repo, err
}

func (g *BitBucketHTTPWrapper) GetCommits(bitBucketURL, workspaceUuid, repoUuid, bitBucketUsername, bitBucketPassword string) (BitBucketRootCommit, error) {
	var err error
	var commits BitBucketRootCommit
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseCommitURL, bitBucketURL, workspaceUuid, repoUuid)
	err = g.get(repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &commits, queryParams, basicFormat)
	// Filter the commits older than three months
	commits = filterDate(commits)
	return commits, err
}

func (g *BitBucketHTTPWrapper) GetRepositories(bitBucketURL, workspaceName, bitBucketUsername, bitBucketPassword string) (BitBucketRootRepoList, error) {
	var err error
	var repos BitBucketRootRepoList
	var queryParams = make(map[string]string)

	repoURL := fmt.Sprintf(bitBucketBaseRepoURL, bitBucketURL, workspaceName)
	err = g.get(repoURL, encodeBitBucketAuth(bitBucketUsername, bitBucketPassword), &repos, queryParams, basicFormat)

	return repos, err
}

func (g *BitBucketHTTPWrapper) get(url, token string, target interface{}, queryParams map[string]string, authFormat string) error {
	var err error

	PrintIfVerbose(fmt.Sprintf("Request to %s", url))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(authFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
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

func filterDate(commits BitBucketRootCommit) BitBucketRootCommit {
	var filteredRootCommits BitBucketRootCommit
	var filteredCommits []BitBucketCommit

	// Get last three months in string and cast it to date
	threeMonthsString := getThreeMonthsTime()
	threeMonths, _ := time.Parse(azureLayoutTime, threeMonthsString)

	for _, commit := range commits.Commits {
		commitDate, _ := time.Parse(time.RFC3339, commit.Date)
		// Check if the commit date occurs after the last three months
		if commitDate.After(threeMonths) {
			filteredCommits = append(filteredCommits, commit)
		}
	}
	// Add the filtered list to the root object
	filteredRootCommits.Commits = filteredCommits
	return filteredRootCommits
}
