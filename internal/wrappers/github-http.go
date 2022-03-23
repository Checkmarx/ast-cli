package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	authorizationHeader = "Authorization"
	apiVersion          = "application/vnd.github.v3+json"
	tokenFormat         = "token %s"
	ownerPlaceholder    = "{owner}"
	repoPlaceholder     = "{repo}"
	orgPlaceholder      = "{org}"
	linkHeaderName      = "Link"
	nextRel             = "next"
	perPageParam        = "per_page"
	perPageValue        = "100"
)

func NewGitHubWrapper() GitHubWrapper {
	return &GitHubHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
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

	return getWithPagination[Repository](g.client, repositoriesURL, map[string]string{})
}

func (g *GitHubHTTPWrapper) GetCommits(repository Repository, queryParams map[string]string) ([]CommitRoot, error) {
	commitsURL := repository.CommitsURL
	commitsURL = commitsURL[:strings.Index(commitsURL, "{")]

	return getWithPagination[CommitRoot](g.client, commitsURL, queryParams)
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
	_, err := get(g.client, url, target, map[string]string{})
	return err
}

func getWithPagination[T CommitRoot | Repository](
	client *http.Client,
	url string,
	queryParams map[string]string,
) ([]T, error) {
	queryParams[perPageParam] = perPageValue

	var pageCollection = make([]T, 0)

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

func collectPage[T CommitRoot | Repository](
	client *http.Client,
	url string,
	queryParams map[string]string,
	pageCollection *[]T,
) (string, error) {
	var holder = make([]T, 0)
	resp, err := get(client, url, &holder, queryParams)
	if err != nil {
		return "", err
	}
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
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(acceptHeader, apiVersion)

	token := viper.GetString(params.SCMTokenFlag)
	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(tokenFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	PrintIfVerbose(fmt.Sprintf("Request to %s", req.URL))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return nil, err
		}
	case http.StatusConflict:
		return nil, nil
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		message := fmt.Sprintf("Code %d %s", resp.StatusCode, string(body))
		return nil, errors.New(message)
	}

	return resp, nil
}
