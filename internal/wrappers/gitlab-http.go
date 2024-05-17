package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/checkmarx/ast-cli/internal/logger"
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
	gitLabProjectsURL         = "%s/%s/projects?membership=true"
	gitLabGroupProjectsURL    = "%s/%s/groups/%s/projects"
	linkHeaderNameGitLab      = "Link"
	nextRelGitLab             = "next"
	perPageParamGitLab        = "per_page"
	perPageValueGitLab        = "100"
	retryLimitGitLab          = 3
)

func NewGitLabWrapper() GitLabWrapper {
	return &GitLabHTTPWrapper{
		client: GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *GitLabHTTPWrapper) GetGitLabProjectsForUser() ([]GitLabProject, error) {
	var err error
	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	getUserProjectsURL := fmt.Sprintf(gitLabProjectsURL, gitLabBaseURL, gitLabAPIVersion)

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
	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)

	encodedProjectPath := url.QueryEscape(gitLabProjectPathWithNameSpace)
	commitsURL := fmt.Sprintf(gitLabCommitURL, gitLabBaseURL, gitLabAPIVersion, encodedProjectPath)

	logger.PrintIfVerbose(fmt.Sprintf("Getting commits for project: %s", gitLabProjectPathWithNameSpace))

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

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	encodedGroupName := url.QueryEscape(gitLabGroupName)

	logger.PrintIfVerbose(fmt.Sprintf("Finding the projects for group: %s", gitLabGroupName))
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
}

func getFromGitLab(
	client *http.Client, requestURL string, target interface{}, queryParams map[string]string,
) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	token := viper.GetString(params.SCMTokenFlag)

	logger.PrintRequest(req)

	resp, err := GetWithQueryParamsAndCustomRequest(client, req, requestURL, token, bearerFormat, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	logger.PrintResponse(resp, true)

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return nil, err
		}
	default:
		body, currentError := io.ReadAll(resp.Body)
		if currentError != nil {
			logger.PrintIfVerbose(currentError.Error())
			return nil, currentError
		}
		message := fmt.Sprintf("Code %d %s", resp.StatusCode, string(body))
		return nil, errors.New(message)
	}
	return resp, nil
}

func fetchWithPagination(
	client *http.Client,
	requestURL string,
	queryParams map[string]string,
) ([]interface{}, error) {
	queryParams[perPageParamGitLab] = perPageValueGitLab

	var pageCollection = make([]interface{}, 0)

	next, err := collectPageForGitLab(client, requestURL, queryParams, &pageCollection)
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
	requestURL string,
	queryParams map[string]string,
	pageCollection *[]interface{},
) (string, error) {
	var holder = make([]interface{}, 0)

	resp, err := getFromGitLab(client, requestURL, &holder, queryParams)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	*pageCollection = append(*pageCollection, holder...)
	nextPageURL := getNextPage(resp)
	return nextPageURL, nil
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
