package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
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
	gitLabGroupProjectsURL    = "%s/%s/groups/%s/projects?per_page=100"
)

func NewGitLabWrapper() GitLabWrapper {
	return &GitLabHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *GitLabHTTPWrapper) GetGitLabProjectsForUser() ([]GitLabProject, error) {
	var err error
	var gitLabProjectList []GitLabProject

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	getUserProjectsURL := fmt.Sprintf(gitLabProjectsURL, gitLabBaseURL, gitLabAPIVersion)
	err = g.get(getUserProjectsURL, &gitLabProjectList, map[string]string{})

	log.Printf("Found %d project(s).", len(gitLabProjectList))
	return gitLabProjectList, err
}

func (g *GitLabHTTPWrapper) GetCommits(
	gitLabProjectPathWithNameSpace string, queryParams map[string]string,
) ([]GitLabCommit, error) {
	var err error
	var commits []GitLabCommit

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)

	encodedProjectPath := url.QueryEscape(gitLabProjectPathWithNameSpace)
	commitsURL := fmt.Sprintf(gitLabCommitURL, gitLabBaseURL, gitLabAPIVersion, encodedProjectPath)

	log.Printf("Getting commits for project: %s", gitLabProjectPathWithNameSpace)
	err = g.get(commitsURL, &commits, queryParams)
	log.Printf("Found %d commit(s).", len(commits))
	return commits, err
}

func (g *GitLabHTTPWrapper) GetGitLabProjects(gitLabGroupName string, queryParams map[string]string) (
	[]GitLabProject, error,
) {
	var err error
	var gitLabProjectList []GitLabProject

	gitLabBaseURL := viper.GetString(params.GitLabURLFlag)
	encodedGroupName := url.QueryEscape(gitLabGroupName)

	log.Printf("Finding the projects for group: %s", gitLabGroupName)
	projectsURL := fmt.Sprintf(gitLabGroupProjectsURL, gitLabBaseURL, gitLabAPIVersion, encodedGroupName)

	err = g.get(projectsURL, &gitLabProjectList, queryParams)
	log.Printf("Found %d project(s).", len(gitLabProjectList))
	return gitLabProjectList, err
}

func (g *GitLabHTTPWrapper) get(requestURL string, target interface{}, queryParams map[string]string) error {
	var err error

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
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
