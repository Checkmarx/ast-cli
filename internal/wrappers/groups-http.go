package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
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

func NewGroupsWrapper(path string) GroupsWrapper {
	return &GroupsHTTPWrapper{path: path}
}

func (g *GroupsHTTPWrapper) Get(groupName string) ([]Group, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	reportPath := fmt.Sprintf("%s?search=%s", g.path, groupName)
	resp, err := SendHTTPRequest(http.MethodGet, reportPath, nil, true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorMsg := ErrorMsg{}
		err = decoder.Decode(&errorMsg)
		if err != nil {
			return nil, errors.Wrapf(err, err.Error())
		}
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
