package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	//"strings"
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
	azureApiVersion      = "6.0"
	azureBaseCommitUrl   = "%s%s/%s/_apis/git/repositories/%s/commits"
	azureBaseReposUrl    = "%s%s/%s/_apis/git/repositories"
	azureBaseProjectsUrl = "%s%s/_apis/projects"
	azureLayoutTime      = "2022-01-01"
	basicFormat          = "Basic %s"
)

func NewAzureWrapper() AzureWrapper {
	return &AzureHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *AzureHTTPWrapper) GetCommits(url string, organizationName string, projectName string, repositoryName string, token string) (AzureRootCommit, error) {
	var err error
	var repository AzureRootCommit
	var queryParams = make(map[string]string)

	commitsUrl := fmt.Sprintf(azureBaseCommitUrl, url, organizationName, projectName, repositoryName)
	queryParams[azureSearchDate] = getThreeMonthsTime()
	queryParams[azureApiVersion] = "6.0"

	err = g.get(commitsUrl, b64.StdEncoding.EncodeToString([]byte(":"+token)), &repository, queryParams, basicFormat)

	return repository, err
}

func (g *AzureHTTPWrapper) GetRepositories(url string, organizationName string, projectName string, token string) (AzureRootRepo, error) {
	var err error
	var repository AzureRootRepo
	var queryParams = make(map[string]string)

	reposUrl := fmt.Sprintf(azureBaseReposUrl, url, organizationName, projectName)
	queryParams["$top"] = "1000000"
	queryParams[azureApiVersion] = "6.0"

	err = g.get(reposUrl, b64.StdEncoding.EncodeToString([]byte(":"+token)), &repository, queryParams, basicFormat)

	return repository, err
}

func (g *AzureHTTPWrapper) GetProjects(url string, organizationName string, token string) (AzureRootProject, error) {
	var err error
	var project AzureRootProject
	var queryParams = make(map[string]string)
	reposUrl := fmt.Sprintf(azureBaseProjectsUrl, url, organizationName)
	queryParams[azureApiVersion] = "6.0"
	err = g.get(reposUrl, b64.StdEncoding.EncodeToString([]byte(":"+token)), &project, queryParams, basicFormat)
	return project, err
}

func (g *AzureHTTPWrapper) get(url string, token string, target interface{}, queryParams map[string]string, authFormat string) error {
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
	case http.StatusNonAuthoritativeInfo:
		err = errors.New("Failed Authentication")
		return err
	case http.StatusForbidden:
		err = errors.New("Failed Authentication")
		return err
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
