package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	b64 "encoding/base64"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type AzureHTTPWrapper struct {
	client *http.Client
}

const (
	azureSearchDate      = "searchCriteria.fromDate"
	azureAPIVersion      = "api-version"
	azureAPIVersionValue = "5.0"
	azureBaseCommitURL   = "%s%s/%s/_apis/git/repositories/%s/commits"
	azureBaseReposURL    = "%s%s/%s/_apis/git/repositories"
	azureBaseProjectsURL = "%s%s/_apis/projects"
	azureTop             = "$top"
	azurePage            = "$skip"
	azureTopValue        = "1000000"
	azureLayoutTime      = "2006-01-02"
	basicFormat          = "Basic %s"
	failedAuth           = "failed Azure Authentication"
	azurePageLenValue    = 100
)

func NewAzureWrapper() AzureWrapper {
	return &AzureHTTPWrapper{
		client: GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *AzureHTTPWrapper) GetCommits(url, organizationName, projectName, repositoryName, token string) (
	AzureRootCommit,
	error,
) {
	var err error
	var repository AzureRootCommit
	var queryParams = make(map[string]string)

	commitsURL := fmt.Sprintf(azureBaseCommitURL, url, organizationName, projectName, repositoryName)
	queryParams[azureSearchDate] = getThreeMonthsTime()
	queryParams[azureAPIVersion] = azureAPIVersionValue
	queryParams[azureTop] = fmt.Sprintf("%s", azurePageLenValue)
	repositories := []AzureRootCommit{}

	azureCommits, err := g.paginateGetter(commitsURL, encodeToken(token), &repository, queryParams, basicFormat)
	if err != nil {
		return repository, err
	}
	bytes, err := json.Marshal(azureCommits)
	if err != nil {
		return repository, err
	}
	err = json.Unmarshal(bytes, &repositories)
	if err != nil {
		return repository, err
	}

	for _, commit := range repositories {
		repository.Commits = append(repository.Commits, commit.Commits...)
	}
	return repository, err
}

func (g *AzureHTTPWrapper) GetRepositories(url, organizationName, projectName, token string) (AzureRootRepo, error) {
	var err error
	var repository AzureRootRepo
	var repositories []AzureRootRepo
	var queryParams = make(map[string]string)

	reposURL := fmt.Sprintf(azureBaseReposURL, url, organizationName, projectName)
	queryParams[azureTop] = fmt.Sprintf("%s", azurePageLenValue)
	queryParams[azureAPIVersion] = azureAPIVersionValue

	azureRepos, err := g.paginateGetter(reposURL, encodeToken(token), &repository, queryParams, basicFormat)
	if err != nil {
		return repository, err
	}

	bytes, err := json.Marshal(azureRepos)
	if err != nil {
		return repository, err
	}

	err = json.Unmarshal(bytes, &repositories)
	if err != nil {
		return repository, err
	}

	for _, commit := range repositories {
		repository.Repos = append(repository.Repos, commit.Repos...)
	}
	return repository, err
}

func (g *AzureHTTPWrapper) GetProjects(url, organizationName, token string) (AzureRootProject, error) {
	var err error
	var project AzureRootProject
	var projects []AzureRootProject
	var queryParams = make(map[string]string)

	reposURL := fmt.Sprintf(azureBaseProjectsURL, url, organizationName)
	queryParams[azureAPIVersion] = azureAPIVersionValue
	queryParams[azureTop] = fmt.Sprintf("%s", azurePageLenValue)

	azureProjects, err := g.paginateGetter(reposURL, encodeToken(token), &project, queryParams, basicFormat)
	if err != nil {
		return project, err
	}

	bytes, err := json.Marshal(azureProjects)
	if err != nil {
		return project, err
	}

	err = json.Unmarshal(bytes, &projects)
	if err != nil {
		return project, err
	}

	for _, commit := range projects {
		project.Projects = append(project.Projects, commit.Projects...)
	}

	return project, err
}

func (g *AzureHTTPWrapper) get(
	url, token string,
	target interface{},
	queryParams map[string]string,
	authFormat string,
) (*http.Response, error) {
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if len(token) > 0 {
		req.Header.Add(AuthorizationHeader, fmt.Sprintf(authFormat, token))
	}

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := g.client.Do(req)

	if err != nil {
		return nil, err
	}

	logger.PrintRequest(req)

	defer func() {
		_ = resp.Body.Close()
	}()

	logger.PrintResponse(resp, true)

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return nil, err
		}
		// State sent when expired token
	case http.StatusNonAuthoritativeInfo:
		err = errors.New(failedAuth)
		return nil, err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New(failedAuth)
		return nil, err
	case http.StatusNotFound:
		// Case the commit/project does not exist in the organization
		return resp, nil
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(body))
	}
	return resp, nil
}

//func paginateGetter(
//	get func(
//		url, token string, target interface{}, queryParams map[string]string, format string) (*http.Response, error)) func(
//	url, token string, target interface{}, queryParams map[string]string, format string) (*[]interface{}, error) {
//	return func(url, token string, target interface{}, queryParams map[string]string, format string) (*[]interface{}, error) {
//		var allTargets []interface{}
//		var currentPage = 0
//		queryParams[azurePage] = strconv.Itoa(currentPage)
//		for {
//			targetCopy := reflect.New(reflect.TypeOf(target).Elem()).Interface()
//
//			resp, err := get(url, token, targetCopy, queryParams, format)
//			if err != nil {
//				return &allTargets, err
//			}
//			allTargets = append(allTargets, targetCopy)
//			if resp.Header.Get("Link") == "" {
//				resp.Body.Close()
//				break
//			}
//			currentPage += 100
//			queryParams[azurePage] = strconv.Itoa(currentPage)
//			resp.Body.Close()
//		}
//		return &allTargets, nil
//	}
//}

func (g *AzureHTTPWrapper) paginateGetter(url, token string, target interface{},
	queryParams map[string]string, format string) (*[]interface{}, error) {
	var allTargets []interface{}
	var currentPage = 0
	queryParams[azurePage] = strconv.Itoa(currentPage)
	for {
		targetCopy := reflect.New(reflect.TypeOf(target).Elem()).Interface()

		resp, err := g.get(url, token, targetCopy, queryParams, format)
		if err != nil {
			return &allTargets, err
		}
		allTargets = append(allTargets, targetCopy)
		if resp.Header.Get("Link") == "" {
			break
		}
		currentPage += azurePageLenValue
		queryParams[azurePage] = strconv.Itoa(currentPage)
	}
	return &allTargets, nil
}

func getThreeMonthsTime() string {
	today := time.Now()
	lastThreeMonths := today.AddDate(0, -3, 0).Format(azureLayoutTime)
	return lastThreeMonths
}

func encodeToken(token string) string {
	return b64.StdEncoding.EncodeToString([]byte(":" + token))
}
