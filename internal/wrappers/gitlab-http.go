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
	client               *http.Client
	repositoryTemplate   string
	organizationTemplate string
}

const (
	gitLabAcceptHeader        = "Accept"
	gitLabAuthorizationHeader = "Authorization"
	gitLabApiVersion          = "api/v4"
	gitLabTokenFormat         = "Bearer %s"
	gitLabCommitUrl           = "%s/%s/projects/%s/repository/commits"
	gitLabProjectsUrl         = "%s/%s/projects?membership=true"
	gitLabGroupSearchUrl      = "%s/%s/groups?all_available=true&search=%s"
	gitLabGroupProjectsUrl    = "%s/%s/groups/%s/projects"
	gitLabUserUrl             = "%s/%s/user"
	gitLabUserProjectsUrl     = "%s/%s/users/%s/projects"
)

func NewGitLabWrapper() GitLabWrapper {
	return &GitLabHTTPWrapper{
		client: getClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *GitLabHTTPWrapper) GetGitLabProjectsForUser() ([]GitLabProject, error) {
	var err error
	var gitLabProjectList []GitLabProject
	var gitLabUsers []GitLabUser

	gitLabBaseURL := viper.GetString(params.URLFlag)

	getUserUrl := fmt.Sprintf(gitLabUserUrl, gitLabBaseURL, gitLabApiVersion)

	log.Printf("Getting user details : %s", getUserUrl)
	err = g.get(getUserUrl, &gitLabUsers, map[string]string{})

	getUserProjectsUrl := fmt.Sprintf(gitLabUserProjectsUrl, gitLabBaseURL, gitLabApiVersion, gitLabUsers[0].ID)

	err = g.get(getUserProjectsUrl, &gitLabProjectList, map[string]string{})

	return gitLabProjectList, err

}

func (g *GitLabHTTPWrapper) GetCommits(gitLabProjectPathWithNameSpace string, queryParams map[string]string) (GitLabRootCommit, error) {
	var err error
	var commits GitLabRootCommit
	var commits2nd []GitLabCommit

	gitLabBaseURL := viper.GetString(params.URLFlag)

	encodedProjectPath := url.QueryEscape(gitLabProjectPathWithNameSpace)
	commitsURL := fmt.Sprintf(gitLabCommitUrl, gitLabBaseURL, gitLabApiVersion, encodedProjectPath)

	log.Printf("Getting commits for project : %s", gitLabProjectPathWithNameSpace)
	log.Printf("Using the url: %s", commitsURL)
	err = g.get(commitsURL, &commits2nd, queryParams)

	return commits, err
}

func (g *GitLabHTTPWrapper) GetGitLabProjects(gitLabGroup GitLabGroup, queryParams map[string]string) ([]GitLabProject, error) {
	var err error
	var gitLabProjectList []GitLabProject
	//var urlSet []string

	gitLabBaseURL := viper.GetString(params.URLFlag)

	// if len(groupList) > 0 {
	// 	for _, group := range groupList {
	// 		urlSet = append(urlSet, fmt.Sprintf(gitLabGroupProjectsUrl, gitLabBaseURL, gitLabApiVersion, group.ID))
	// 	}

	// } else {
	// 	urlSet = append(urlSet, fmt.Sprintf(gitLabProjectsUrl, gitLabBaseURL, gitLabApiVersion))
	// }

	// for _, url := range urlSet {
	// 	err = g.get(url, &gitLabProjectList, queryParams)
	// }
	var url string
	if gitLabGroup == (GitLabGroup{}) {
		url = fmt.Sprintf(gitLabProjectsUrl, gitLabBaseURL, gitLabApiVersion)
	} else {
		url = fmt.Sprintf(gitLabGroupProjectsUrl, gitLabBaseURL, gitLabApiVersion, gitLabGroup.ID)
	}

	//groupSearchUrl := fmt.Sprintf(gitLabGroupProjectsUrl, gitLabBaseURL, gitLabApiVersion, gitLabGroup.ID)
	log.Printf("Using the url: %s", url)
	err = g.get(url, &gitLabProjectList, queryParams)

	return gitLabProjectList, err
}

func (g *GitLabHTTPWrapper) GetGitLabGroups(groupName string) ([]GitLabGroup, error) {
	var err error
	var gitLabGroupList []GitLabGroup

	gitLabBaseURL := viper.GetString(params.URLFlag)
	gitLabGroupUrl := fmt.Sprintf(gitLabGroupSearchUrl, gitLabBaseURL, gitLabApiVersion, groupName)

	log.Printf("Getting the details for group : %s", groupName)
	log.Printf("Using the url: %s", gitLabGroupUrl)
	err = g.get(gitLabGroupUrl, &gitLabGroupList, map[string]string{})

	return gitLabGroupList, err
}

func (g *GitLabHTTPWrapper) get(url string, target interface{}, queryParams map[string]string) error {
	var err error

	PrintIfVerbose(fmt.Sprintf("Request to %s", url))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	token := viper.GetString(params.SCMTokenFlag)
	if len(token) > 0 {
		req.Header.Add(authorizationHeader, fmt.Sprintf(gitLabTokenFormat, token))
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

	log.Printf("Response Status : %d", resp.StatusCode)
	//b, err := io.ReadAll(resp.Body)
	// log.Printf("Response Body : %s", string(b))
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
