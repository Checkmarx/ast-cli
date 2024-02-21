package wrappers

type CreateAssignment struct {
	EntityID     string   `json:"entityID"`
	EntityType   string   `json:"entityType"`
	EntityName   string   `json:"entityName"`
	EntityRoles  []string `json:"entityRoles"`
	ResourceID   string   `json:"resourceID"`
	ResourceType string   `json:"resourceType"`
	ResourceName string   `json:"resourceName"`
}

type AccessManagementWrapper interface {
	CreateGroupsAssignment(projectId string, projectName string, groups []*Group) error
	GetGroups(projectId string) ([]*Group, error)
}
