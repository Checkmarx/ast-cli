package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// APIs
	createAssignmentPath = ""
	entitiesForPath      = "entities-for"

	// EntityTypes
	groupEntityType     = "group"
	projectResourceType = "project"
)

type HasAccessResponse struct {
	HasAccess bool `json:"accessGranted"`
}

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
		assignment := AssignmentPayload{
			EntityID:   group.ID,
			EntityType: groupEntityType,
			//EntityRoles:  nil, // be used in the access-management phase 2
			ResourceID:   projectID,
			ResourceType: projectResourceType,
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
		logger.PrintfIfVerbose("group '%s' assignment for project %s created", group.Name, projectName)
		resp.Body.Close()
	}
	logger.PrintIfVerbose("Groups assignment created successfully")
	return nil
}

func (a *AccessManagementHTTPWrapper) GetGroups(projectID string) ([]*Group, error) {
	path := fmt.Sprintf("%s/%s?resource-id=%s&resource-type=project", a.path, entitiesForPath, projectID)
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, true, a.clientTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get groups")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to get groups, status code: %d", resp.StatusCode)
	}
	var assignments []*AssignmentResponse
	var groups []*Group
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&assignments)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse response body")
	}
	for _, assignment := range assignments {
		if assignment.EntityType == groupEntityType {
			group := &Group{
				ID:   assignment.EntityID,
				Name: assignment.EntityName,
			}
			groups = append(groups, group)
		}
	}
	return groups, nil
}
