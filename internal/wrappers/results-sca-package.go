package wrappers

type VulnerabilitiesRisks struct {
	VulnerabilitiesRisks Items `json:"vulnerabilitiesRisksByScanId"`
}

type Items struct {
	Items      []ScaVulnerability `json:"items"`
	TotalCount int                `json:"totalCount"`
}

type ScaVulnerability struct {
	ID       string `json:"packageId,omitempty"`
	Relation string `json:"relation,omitempty"`
}

type ScaPackageCollection struct {
	ID                  string             `json:"id,omitempty"`
	FixLink             string             `json:"fixLink,omitempty"`
	BestPackageLink     string             `json:"bestPackageLink,omitempty"`
	Locations           []*string          `json:"locations,omitempty"`
	DependencyPathArray [][]DependencyPath `json:"dependencyPaths,omitempty"`
	Outdated            bool               `json:"outdated,omitempty"`
	SupportsQuickFix    bool               `json:"supportsQuickFix"`
	IsDirectDependency  bool               `json:"isDirectDependency"`
	TypeOfDependency    string             `json:"typeOfDependency"`
}

type DependencyPath struct {
	ID               string    `json:"id,omitempty"`
	Name             string    `json:"name,omitempty"`
	Version          string    `json:"version,omitempty"`
	IsResolved       bool      `json:"isResolved,omitempty"`
	IsDevelopment    bool      `json:"isDevelopment,omitempty"`
	Locations        []*string `json:"locations,omitempty"`
	SupportsQuickFix bool      `json:"supportsQuickFix,omitempty"`
}

type ScaTypeCollection struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

type GraphQLVulnerabilityRisks struct {
	Data   VulnerabilitiesRisks `json:"data"`
	Errors []GraphQLError       `json:"errors"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}
