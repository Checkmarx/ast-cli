package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/tomnomnom/linkheader"
)

type GitHubHTTPWrapper struct {
	client               *http.Client
	repositoryTemplate   string
	organizationTemplate string
}

const (
	acceptHeader        = "Accept"
	AuthorizationHeader = "Authorization"
	apiVersion          = "application/vnd.github.v3+json"
	tokenFormat         = "token %s"
	ownerPlaceholder    = "{owner}"
	repoPlaceholder     = "{repo}"
	orgPlaceholder      = "{org}"
	linkHeaderName      = "Link"
	nextRel             = "next"
	perPageParam        = "per_page"
	perPageValue        = "100"
	retryLimit          = 3
)

func NewGitHubWrapper() GitHubWrapper {
	return &GitHubHTTPWrapper{
		client: GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *GitHubHTTPWrapper) GetOrganization(organizationName string) (Organization, error) {
	var err error
	var organization Organization

	organizationTemplate, err := g.getOrganizationTemplate()
	if err != nil {
		return organization, err
	}
	organizationURL := strings.ReplaceAll(organizationTemplate, orgPlaceholder, organizationName)

	err = g.get(organizationURL, &organization)

	return organization, err
}

func (g *GitHubHTTPWrapper) GetRepository(organizationName, repositoryName string) (Repository, error) {
	var err error
	var repository Repository

	repositoryURL, err := g.getRepositoryTemplate()
	if err != nil {
		return repository, err
	}
	repositoryURL = strings.ReplaceAll(repositoryURL, ownerPlaceholder, organizationName)
	repositoryURL = strings.ReplaceAll(repositoryURL, repoPlaceholder, repositoryName)

	err = g.get(repositoryURL, &repository)

	return repository, err
}

func (g *GitHubHTTPWrapper) GetRepositories(organization Organization) ([]Repository, error) {
	repositoriesURL := organization.RepositoriesURL

	pages, err := getWithPagination(g.client, repositoriesURL, map[string]string{})
	if err != nil {
		return nil, err
	}

	castedPages := make([]Repository, 0)
	for _, e := range pages {
		marshal, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		holder := Repository{}
		err = json.Unmarshal(marshal, &holder)
		if err != nil {
			return nil, err
		}
		castedPages = append(castedPages, holder)
	}

	return castedPages, nil
}

func (g *GitHubHTTPWrapper) GetCommits(repository Repository, queryParams map[string]string) ([]CommitRoot, error) {
	commitsURL := repository.CommitsURL
	index := strings.Index(commitsURL, "{")
	if index < 0 {
		return nil, errors.Errorf("Unable to collect commits URL for repository %s", repository.FullName)
	}
	commitsURL = commitsURL[:index]

	pages, err := getWithPagination(g.client, commitsURL, queryParams)
	if err != nil {
		return nil, err
	}

	castedPages := make([]CommitRoot, 0)
	for _, e := range pages {
		marshal, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		holder := CommitRoot{}
		err = json.Unmarshal(marshal, &holder)
		if err != nil {
			return nil, err
		}
		castedPages = append(castedPages, holder)
	}

	return castedPages, nil
}

func (g *GitHubHTTPWrapper) getOrganizationTemplate() (string, error) {
	var err error

	if g.organizationTemplate == "" {
		err = g.getTemplates()
	}

	return g.organizationTemplate, err
}

func (g *GitHubHTTPWrapper) getRepositoryTemplate() (string, error) {
	var err error

	if g.repositoryTemplate == "" {
		err = g.getTemplates()
	}

	return g.repositoryTemplate, err
}

func (g *GitHubHTTPWrapper) getTemplates() error {
	var err error
	var rootAPIResponse rootAPI

	baseURL := viper.GetString(params.URLFlag)
	err = g.get(baseURL, &rootAPIResponse)

	g.organizationTemplate = rootAPIResponse.OrganizationURL
	g.repositoryTemplate = rootAPIResponse.RepositoryURL

	return err
}

func (g *GitHubHTTPWrapper) get(url string, target interface{}) error {
	resp, err := get(g.client, url, target, map[string]string{})
	if err != nil {
		defer func() {
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
	}
	return err
}

func getWithPagination(
	client *http.Client,
	url string,
	queryParams map[string]string,
) ([]interface{}, error) {
	queryParams[perPageParam] = perPageValue

	var pageCollection = make([]interface{}, 0)

	next, err := collectPage(client, url, queryParams, &pageCollection)
	if err != nil {
		return nil, err
	}

	for next != "" {
		next, err = collectPage(client, next, map[string]string{}, &pageCollection)
		if err != nil {
			return nil, err
		}
	}

	return pageCollection, nil
}

func collectPage(
	client *http.Client,
	url string,
	queryParams map[string]string,
	pageCollection *[]interface{},
) (string, error) {
	var holder = make([]interface{}, 0)

	resp, err := get(client, url, &holder, queryParams)
	if err != nil {
		return "", err
	}

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	*pageCollection = append(*pageCollection, holder...)
	next := getNextPageLink(resp)

	return next, nil
}

func getNextPageLink(resp *http.Response) string {
	if resp != nil {
		linkHeader := resp.Header[linkHeaderName]
		if len(linkHeader) > 0 {
			links := linkheader.Parse(linkHeader[0])
			for _, link := range links {
				if link.Rel == nextRel {
					return link.URL
				}
			}
		}
	}
	return ""
}

func get(client *http.Client, url string, target interface{}, queryParams map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add(acceptHeader, apiVersion)
	token := viper.GetString(params.SCMTokenFlag)
	logger.PrintRequest(req)
	resp, err := GetWithQueryParamsAndCustomRequest(client, req, url, token, tokenFormat, queryParams)
	if err != nil {
		return nil, err
	}
	resp, err = handleRateLimit(resp, client, req, url, token, queryParams)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	logger.PrintResponse(resp, true)

	switch resp.StatusCode {
	case http.StatusOK:
		logger.PrintIfVerbose(fmt.Sprintf("Request to URL %s OK", req.URL))
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return nil, err
		}
	case http.StatusConflict:
		logger.PrintIfVerbose(fmt.Sprintf("Found empty repository in %s", req.URL))
		return resp, nil
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
			return nil, err
		}
		message := fmt.Sprintf("Code %d %s", resp.StatusCode, string(body))
		return nil, errors.New(message)
	}
	return resp, nil
}

func handleRateLimit(resp *http.Response, client *http.Client, req *http.Request, url, token string, queryParams map[string]string) (*http.Response, error) {
	if resp.StatusCode == http.StatusForbidden {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		if remaining == "0" && reset != "" {
			resetUnix, err := strconv.ParseInt(reset, 10, 64)
			if err == nil {
				waitDuration := time.Until(time.Unix(resetUnix, 0))
				if waitDuration > 0 {
					time.Sleep(waitDuration)
					return GetWithQueryParamsAndCustomRequest(client, req, url, token, tokenFormat, queryParams) // Indicate to retry
				}
			} else {
				return resp, err
			}
		}
	}
	return resp, nil // Not rate limited
}
