package wrappers

import (
	"math/big"
)

type ResultsWrapper interface {
	GetByScanID(scanID string, limit, offset uint64) ([]ResultResponseModel, *ResultError, error)
}

type ResultError struct {
	Code    int32       `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ResultNode struct {
	// Column position of the node
	Column int32 `json:"column,omitempty"`
	// Full file name of the containing source file
	FileName string `json:"fileName,omitempty"`
	// FQN of the node
	FullName string `json:"fullName,omitempty"`
	// Length of the node
	Length int32 `json:"length,omitempty"`
	// Line position of the node
	Line int32 `json:"line,omitempty"`
	// Line position of the containing method
	MethodLine int32 `json:"methodLine,omitempty"`
	// node name
	Name string `json:"name,omitempty"`
	// ID of node
	NodeID int32 `json:"nodeID,omitempty"`
	// node DomType
	DomType string `json:"domType,omitempty"`
	// ID of the customer tenant
	NodeSystemID string `json:"nodeSystemID,omitempty"`
}

type ResultResponseModel struct {
	// Query ID
	QueryID int32 `json:"queryID,omitempty"`
	// Query name
	QueryName string `json:"queryName,omitempty"`
	// Query group; sperate by ':'
	GroupName string `json:"groupName,omitempty"`
	// Severity of result
	Severity string `json:"severity,omitempty"`
	// Common Weakness Enumeration ID
	CweID int32 `json:"cweID,omitempty"`
	// ID of the path. changes from scan to scan.
	PathID int32 `json:"pathID,omitempty"`
	// ID of the Similarity feature (Indicator to identify a result by its first and last nodes)
	SimilarityID int32 `json:"similarityID,omitempty"`
	// Same as similarityID but can change in the future (SAST feature)
	UniqueID int32 `json:"uniqueID,omitempty"`
	// Confidence Level of the exsitin of the result
	ConfidenceLevel big.Float `json:"confidenceLevel,omitempty"`

	Nodes []ResultNode `json:"nodes,omitempty"`
	// ID of the customer tenant
	TenantID string `json:"tenantID,omitempty"`
	// ID of the scan
	ScanID string `json:"scanID,omitempty"`
	// Creation date of the result
	CreatedAt string `json:"createdAt,omitempty"`

	Classification string `json:"classification,omitempty"`
	// Groups arrays
	Groups []string `json:"groups,omitempty"`
	// ID of the customer tenant
	PathSystemID string `json:"pathSystemID,omitempty"`
	// ID created from queryMetaInfo + similarityID + files name
	PathSystemIDBySimiAndFilesPaths string `json:"pathSystemIDBySimiAndFilesPaths,omitempty"`
	// enum of the current state(new,old,fixed)
	Status string `json:"status,omitempty"`
	// TBD
	MetadataJSON string `json:"metadataJSON,omitempty"`
	// TBD
	ExtraJSON string `json:"extraJSON,omitempty"`
}
