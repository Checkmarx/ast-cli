package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
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
	unauthorized         = "unauthorized: verify if the organization you provided is correct"
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
	var rootCommit AzureRootCommit
	var pages []AzureRootCommit
	var queryParams = make(map[string]string)

	commitsURL := fmt.Sprintf(azureBaseCommitURL, url, organizationName, projectName, repositoryName)
	queryParams[azureSearchDate] = getThreeMonthsTime()
	queryParams[azureAPIVersion] = azureAPIVersionValue
	queryParams[azureTop] = fmt.Sprintf("%d", azurePageLenValue)

	err = g.paginateGetter(commitsURL, encodeToken(token), &AzureRootCommit{}, &pages, queryParams, basicFormat)
	if err != nil {
		return rootCommit, err
	}

	for _, commitPage := range pages {
		rootCommit.Commits = append(rootCommit.Commits, commitPage.Commits...)
	}

	return rootCommit, err
}

func (g *AzureHTTPWrapper) GetRepositories(url, organizationName, projectName, token string) (AzureRootRepo, error) {
	var err error
	var rootRepo AzureRootRepo
	var pages []AzureRootRepo
	var queryParams = make(map[string]string)

	reposURL := fmt.Sprintf(azureBaseReposURL, url, organizationName, projectName)
	queryParams[azureTop] = fmt.Sprintf("%d", azurePageLenValue)
	queryParams[azureAPIVersion] = azureAPIVersionValue

	err = g.paginateGetter(reposURL, encodeToken(token), &AzureRootRepo{}, &pages, queryParams, basicFormat)
	if err != nil {
		return rootRepo, err
	}

	for _, repositoryPage := range pages {
		rootRepo.Repos = append(rootRepo.Repos, repositoryPage.Repos...)
	}

	return rootRepo, err
}

func (g *AzureHTTPWrapper) GetProjects(url, organizationName, token string) (AzureRootProject, error) {
	var err error
	var rootProject AzureRootProject
	var pages []AzureRootProject
	var queryParams = make(map[string]string)

	reposURL := fmt.Sprintf(azureBaseProjectsURL, url, organizationName)
	queryParams[azureAPIVersion] = azureAPIVersionValue
	queryParams[azureTop] = fmt.Sprintf("%d", azurePageLenValue)

	err = g.paginateGetter(reposURL, encodeToken(token), &AzureRootProject{}, &pages, queryParams, basicFormat)
	if err != nil {
		return rootProject, err
	}

	for _, projectPage := range pages {
		rootProject.Projects = append(rootProject.Projects, projectPage.Projects...)
	}

	return rootProject, err
}

func (g *AzureHTTPWrapper) get(
	url, token string,
	target interface{},
	queryParams map[string]string,
	authFormat string,
) (bool, error) {
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
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
		return false, err
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
			return false, err
		}
		// State sent when expired token
	case http.StatusNonAuthoritativeInfo:
		err = errors.New(failedAuth)
		return false, err
		// State sent when no token is provided
	case http.StatusForbidden:
		err = errors.New(failedAuth)
		return false, err
	case http.StatusUnauthorized:
		return false, errors.New(unauthorized)
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		return false, errors.Errorf("%s - %s", string(body), resp.Status)
	}
	headerLink := resp.Header.Get("Link")
	continuationToken := resp.Header.Get("X-Ms-Continuationtoken")
	return headerLink != "" || continuationToken != "", nil
}

func (g *AzureHTTPWrapper) paginateGetter(url, token string, target interface{}, slice interface{}, queryParams map[string]string, format string) error {
	var currentPage = 0
	for {
		queryParams[azurePage] = fmt.Sprintf("%d", currentPage)
		hasNextPage, err := g.get(url, token, target, queryParams, format)
		if err != nil {
			return err
		}

		slicePtr := reflect.ValueOf(slice)
		sliceValue := slicePtr.Elem()
		sliceValue.Set(reflect.Append(sliceValue, reflect.ValueOf(target).Elem()))

		target = reflect.New(reflect.TypeOf(target).Elem()).Interface()

		if !hasNextPage {
			break
		}

		currentPage += azurePageLenValue
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
