package wrappers

type SastMetadataWrapper interface {
	GetSastMetadataByIDs(params map[string]string) (*SastMetadataModel, error)
}

type SastMetadataModel struct {
	TotalCount int      `json:"totalCount"`
	Scans      []Scans  `json:"scans"`
	Missing    []string `json:"missing"`
}
type Scans struct {
	ScanID                  string `json:"scanId,omitempty"`
	ProjectID               string `json:"projectId,omitempty"`
	Loc                     int    `json:"loc,omitempty"`
	FileCount               int    `json:"fileCount,omitempty"`
	IsIncremental           bool   `json:"isIncremental,omitempty"`
	IsIncrementalCanceled   bool   `json:"isIncrementalCanceled,omitempty"`
	IncrementalCancelReason string `json:"incrementalCancelReason,omitempty"`
	BaseID                  string `json:"baseId,omitempty"`
	AddedFilesCount         int    `json:"addedFilesCount,omitempty"`
	ChangedFilesCount       int    `json:"changedFilesCount,omitempty"`
	DeletedFilesCount       int    `json:"deletedFilesCount,omitempty"`
	QueryPreset             string `json:"queryPreset,omitempty"`
}
