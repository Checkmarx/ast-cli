package wrappers

import (
	"encoding/json"
	"net/http"
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToGetGroups = "Failed to parse list response"
)

type GroupsHTTPWrapper struct {
	path string
}

type Group struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func NewHTTPGroupsWrapper(path string) GroupsWrapper {
	return &GroupsHTTPWrapper{path: path}
}

func (g *GroupsHTTPWrapper) Get(groupName string) ([]Group, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	tenant := viper.GetString(commonParams.TenantKey)
	tenantPath := strings.Replace(g.path, "organization", strings.ToLower(tenant), 1)
	groupMap := make(map[string]string)
	groupMap["groupName"] = groupName
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, tenantPath, groupMap, nil, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorMsg := GetError(decoder)
		return nil, errors.Errorf("%s: CODE: %d, %s", failedToGetGroups, errorMsg.Code, errorMsg.Message)
	case http.StatusOK:
		var groups []Group
		err = decoder.Decode(&groups)
		if err != nil {
			return nil, errors.Wrapf(err, failedToGetGroups)
		}
		return groups, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
