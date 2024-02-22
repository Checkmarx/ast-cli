package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

const (
	// APIs
	createAssignmentPath = "/"
	entitiesForPath      = "/entities-for"
	// EntityTypes
	groupEntityType     = "group"
	projectResourceType = "project"
)

type AccessManagementHTTPWrapper struct {
	path          string
	clientTimeout uint
}

func NewAccessManagementHTTPWrapper(path string) AccessManagementWrapper {
	return &AccessManagementHTTPWrapper{
		path:          path,
		clientTimeout: viper.GetUint(commonParams.ClientTimeoutKey),
	}
}
func (a *AccessManagementHTTPWrapper) CreateGroupsAssignment(projectID, projectName string, groups []*Group) error {
	var resp *http.Response
	for _, group := range groups {
		assignment := CreateAssignment{
			EntityID:     group.ID,
			EntityType:   groupEntityType,
			EntityName:   group.Name,
			EntityRoles:  nil, // be used in the access-management phase 2
			ResourceID:   projectID,
			ResourceType: projectResourceType,
			ResourceName: projectName,
		}
		params, err := json.Marshal(assignment)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse request body")
		}
		path := fmt.Sprintf("%s/%s", a.path, createAssignmentPath)
		resp, err = SendHTTPRequestWithJSONContentType(http.MethodPost, path, bytes.NewBuffer(params), true, a.clientTimeout)
		if err != nil {
			return errors.Wrapf(err, "Failed to create groups assignment")
		}

	}
	defer resp.Body.Close()
	return nil
}

func (a *AccessManagementHTTPWrapper) GetGroups(projectID string) ([]*Group, error) {
	log.Println("Getting groups")
	path := fmt.Sprintf("%s/%s?resource-id=%s&entity-types=group", a.path, entitiesForPath, projectID)
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, true, a.clientTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get groups")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to get groups, status code: %d", resp.StatusCode)
	}
	var groups []*Group
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&groups)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse response body")
	}
	return groups, nil
}
