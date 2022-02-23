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

	err = g.get(organizationURL, &organization, map[string]string{})

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

	err = g.get(repositoryURL, &repository, map[string]string{})

	return repository, err
}

func (g *GitHubHTTPWrapper) GetRepositories(organization Organization) ([]Repository, error) {
	var err error
	var repositories []Repository

	repositoriesURL := organization.RepositoriesURL

	err = g.get(repositoriesURL, &repositories, map[string]string{})

	return repositories, err
}

func (g *GitHubHTTPWrapper) GetCommits(repository Repository, queryParams map[string]string) ([]Commit, error) {
	var err error
	var commits []Commit

	commitsURL := repository.CommitsURL
	commitsURL = commitsURL[:strings.Index(commitsURL, "{")]

	err = g.get(commitsURL, &commits, queryParams)

	return commits, err
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

	baseURL := viper.GetString(params.GitHubURLFlag)
	err = g.get(baseURL, &rootAPIResponse, map[string]string{})

	g.organizationTemplate = rootAPIResponse.OrganizationURL
	g.repositoryTemplate = rootAPIResponse.RepositoryURL

	return err
}

func (g *GitHubHTTPWrapper) get(url string, target interface{}, queryParams map[string]string) error {
	PrintIfVerbose(fmt.Sprintf("Request to %s", url))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add(acceptHeader, apiVersion)

	token := viper.GetString(params.SCMTokenFlag)
	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(tokenFormat, token))
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
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	return nil
}
