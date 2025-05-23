package wrappers

type AssignmentResponse struct {
	EntityID     string   `json:"entityID"`
	EntityType   string   `json:"entityType"`
	EntityName   string   `json:"entityName"`
	EntityRoles  []string `json:"entityRoles"`
	ResourceID   string   `json:"resourceID"`
	ResourceType string   `json:"resourceType"`
	ResourceName string   `json:"resourceName"`
}

type AccessManagementWrapper interface {
	CreateGroupsAssignment(projectID, projectName string, groups []*Group) error
	GetGroups(projectID string) ([]*Group, error)
	HasEntityAccessToGroups(groupIDs []string) (bool, error)
}

type AssignmentPayload struct {
	EntityID     string        `json:"entityID"`
	EntityType   string        `json:"entityType"`
	EntityRoles  []interface{} `json:"entityRoles"`
	ResourceType string        `json:"resourceType"`
	ResourceID   string        `json:"resourceID"`
}
