package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/tomnomnom/linkheader"
)

type GitLabHTTPWrapper struct {
	client *http.Client
}

const (
	gitLabAuthorizationHeader = "Authorization"
	gitLabAPIVersion          = "api/v4"
	gitLabTokenFormat         = "Bearer %s"
	gitLabCommitURL           = "%s/%s/projects/%s/repository/commits"
	gitLabProjectsURL         = "%s/%s/projects?per_page=100&membership=true"
	gitLabGroupProjectsURL    = "%s/%s/groups/%s/projects" //?per_page=100"
	linkHeaderNameGitLab      = "Link"
	nextRelGitLab             = "next"
	perPageParamGitLab        = "per_page"
	perPageValueGitLab        = "100"
	retryLimitGitLab          = 3
)

func NewGitLabWrapper() GitLabWrapper {
	return &GitLabHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *GitLabHTTPWrapper) GetGitLabProjectsForUser() ([]GitLabProject, error) {
	var err error
	//var gitLabProjectList []GitLabProject

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	getUserProjectsURL := fmt.Sprintf(gitLabProjectsURL, gitLabBaseURL, gitLabAPIVersion)
	//err = g.get(getUserProjectsURL, &gitLabProjectList, map[string]string{})

	//PrintIfVerbose(fmt.Sprintf("Found %d project(s).", len(gitLabProjectList)))
	//return gitLabProjectList, err

	pages, err := fetchWithPagination(g.client, getUserProjectsURL, map[string]string{})
	if err != nil {
		return nil, err
	}
	castedPages := make([]GitLabProject, 0)
	for _, e := range pages {
		marshal, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		holder := GitLabProject{}
		err = json.Unmarshal(marshal, &holder)
		if err != nil {
			return nil, err
		}
		castedPages = append(castedPages, holder)
	}

	return castedPages, nil
}

func (g *GitLabHTTPWrapper) GetCommits(
	gitLabProjectPathWithNameSpace string, queryParams map[string]string,
) ([]GitLabCommit, error) {
	var err error
	//var commits []GitLabCommit

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)

	encodedProjectPath := url.QueryEscape(gitLabProjectPathWithNameSpace)
	commitsURL := fmt.Sprintf(gitLabCommitURL, gitLabBaseURL, gitLabAPIVersion, encodedProjectPath)

	PrintIfVerbose(fmt.Sprintf("Getting commits for project: %s", gitLabProjectPathWithNameSpace))
	//err = g.get(commitsURL, &commits, queryParams)
	//PrintIfVerbose(fmt.Sprintf("Found %d commit(s).", len(commits)))
	//return commits, err

	pages, err := fetchWithPagination(g.client, commitsURL, queryParams)
	if err != nil {
		return nil, err
	}
	castedPages := make([]GitLabCommit, 0)
	for _, e := range pages {
		marshal, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		holder := GitLabCommit{}
		err = json.Unmarshal(marshal, &holder)
		if err != nil {
			return nil, err
		}
		castedPages = append(castedPages, holder)
	}

	return castedPages, nil
}

func (g *GitLabHTTPWrapper) GetGitLabProjects(gitLabGroupName string, queryParams map[string]string) (
	[]GitLabProject, error,
) {
	var err error
	//var gitLabProjectList []GitLabProject

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	encodedGroupName := url.QueryEscape(gitLabGroupName)

	PrintIfVerbose(fmt.Sprintf("Finding the projects for group: %s", gitLabGroupName))
	projectsURL := fmt.Sprintf(gitLabGroupProjectsURL, gitLabBaseURL, gitLabAPIVersion, encodedGroupName)

	pages, err := fetchWithPagination(g.client, projectsURL, queryParams)
	if err != nil {
		return nil, err
	}
	castedPages := make([]GitLabProject, 0)
	for _, e := range pages {
		marshal, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		holder := GitLabProject{}
		err = json.Unmarshal(marshal, &holder)
		if err != nil {
			return nil, err
		}
		castedPages = append(castedPages, holder)
	}

	return castedPages, nil

	//err = g.get(projectsURL, &gitLabProjectList, queryParams)
	//PrintIfVerbose(fmt.Sprintf("Found %d project(s).", len(gitLabProjectList)))
	//return gitLabProjectList, err
}

func getFromGitLab(
	client *http.Client, requestURL string, target interface{}, queryParams map[string]string,
) (*http.Response, error) {
	var err error
	var count uint8

	for count < retryLimitGitLab {
		var currentError error
		req, currentError := http.NewRequest(http.MethodGet, requestURL, http.NoBody)
		if currentError != nil {
			return nil, currentError
		}

		token := viper.GetString(params.SCMTokenFlag)
		if len(token) > 0 {
			req.Header.Add(gitLabAuthorizationHeader, fmt.Sprintf(gitLabTokenFormat, token))
		}
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
		PrintIfVerbose(fmt.Sprintf("Request to %s", req.URL))
		resp, currentError := client.Do(req)
		if currentError != nil {
			count++
			PrintIfVerbose(fmt.Sprintf("Request to %s dropped, retrying", req.URL))
			err = currentError
			continue
		}

		switch resp.StatusCode {
		case http.StatusOK:
			currentError = json.NewDecoder(resp.Body).Decode(target)
			closeBody(resp)
			if currentError != nil {
				return nil, currentError
			}
		default:
			body, currentError := io.ReadAll(resp.Body)
			closeBody(resp)
			if currentError != nil {
				PrintIfVerbose(currentError.Error())
				return nil, currentError
			}
			message := fmt.Sprintf("Code %d %s", resp.StatusCode, string(body))
			return nil, errors.New(message)
		}
		return resp, nil
	}

	return nil, err
}

func (g *GitLabHTTPWrapper) getFromGitLab(url string, target interface{}) error {
	resp, err := get(g.client, url, target, map[string]string{})

	closeResponseBody(resp)

	return err
}

func fetchWithPagination(
	client *http.Client,
	url string,
	queryParams map[string]string,
) ([]interface{}, error) {
	queryParams[perPageParamGitLab] = perPageValueGitLab

	var pageCollection = make([]interface{}, 0)

	next, err := collectPageForGitLab(client, url, queryParams, &pageCollection)
	if err != nil {
		return nil, err
	}

	for next != "" {
		next, err = collectPageForGitLab(client, next, map[string]string{}, &pageCollection)
		if err != nil {
			return nil, err
		}
	}

	return pageCollection, nil
}

func collectPageForGitLab(
	client *http.Client,
	url string,
	queryParams map[string]string,
	pageCollection *[]interface{},
) (string, error) {
	var holder = make([]interface{}, 0)

	resp, err := getFromGitLab(client, url, &holder, queryParams)
	if err != nil {
		return "", err
	}
	defer closeResponseBody(resp)

	*pageCollection = append(*pageCollection, holder...)
	nextPageUrl := getNextPage(resp)
	return nextPageUrl, nil
}

func getNextPage(resp *http.Response) string {
	if resp != nil {
		linkHeader := resp.Header[linkHeaderNameGitLab]
		if len(linkHeader) > 0 {
			links := linkheader.Parse(linkHeader[0])
			for _, link := range links {
				if link.Rel == nextRelGitLab {
					return link.URL
				}
			}
		}
	}
	return ""
}

func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
