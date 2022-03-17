package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	b64 "encoding/base64"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type AzureHTTPWrapper struct {
	client               *http.Client
	repositoryTemplate   string
	organizationTemplate string
}

const (
	azureSearchDate      = "searchCriteria.fromDate"
	azureApiVersion      = "api-version"
	azureApiVersionValue = "6.0"
	azureBaseCommitUrl   = "%s%s/%s/_apis/git/repositories/%s/commits"
	azureBaseReposUrl    = "%s%s/%s/_apis/git/repositories"
	azureBaseProjectsUrl = "%s%s/_apis/projects"
	azureTop             = "$top"
	azureTopValue        = "1000000"
	azureLayoutTime      = "2006-01-02"
	basicFormat          = "Basic %s"
	failedAuth           = "Failed Azure Authentication"
)

func NewAzureWrapper() AzureWrapper {
	return &AzureHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *AzureHTTPWrapper) GetCommits(url, organizationName, projectName, repositoryName, token string) (AzureRootCommit, error) {
	var err error
	var repository AzureRootCommit
	var queryParams = make(map[string]string)

	commitsUrl := fmt.Sprintf(azureBaseCommitUrl, url, organizationName, projectName, repositoryName)
	queryParams[azureSearchDate] = getThreeMonthsTime()
	queryParams[azureApiVersion] = azureApiVersionValue

	err = g.get(commitsUrl, encodeToken(token), &repository, queryParams, basicFormat)

	return repository, err
}

func (g *AzureHTTPWrapper) GetRepositories(url, organizationName, projectName, token string) (AzureRootRepo, error) {
	var err error
	var repository AzureRootRepo
	var queryParams = make(map[string]string)

	reposUrl := fmt.Sprintf(azureBaseReposUrl, url, organizationName, projectName)
	queryParams[azureTop] = azureTopValue
	queryParams[azureApiVersion] = azureApiVersionValue

	err = g.get(reposUrl, encodeToken(token), &repository, queryParams, basicFormat)

	return repository, err
}

func (g *AzureHTTPWrapper) GetProjects(url, organizationName, token string) (AzureRootProject, error) {
	var err error
	var project AzureRootProject
	var queryParams = make(map[string]string)

	reposUrl := fmt.Sprintf(azureBaseProjectsUrl, url, organizationName)
	queryParams[azureApiVersion] = azureApiVersionValue

	err = g.get(reposUrl, encodeToken(token), &project, queryParams, basicFormat)

	return project, err
}

func (g *AzureHTTPWrapper) get(url, token string, target interface{}, queryParams map[string]string, authFormat string) error {
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
	case http.StatusNonAuthoritativeInfo:
		err = errors.New(failedAuth)
		return err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New(failedAuth)
		return err
	case http.StatusNotFound:
		// Case the commit/project does not exist in the organization
		return nil
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func getThreeMonthsTime() string {
	today := time.Now()
	lastThreeMonths := today.AddDate(0, -3, 0).Format(azureLayoutTime)
	return lastThreeMonths
}

func encodeToken(token string) string {
	return b64.StdEncoding.EncodeToString([]byte(":" + token))
}
