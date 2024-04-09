package wrappers

import (
	"encoding/json"
	"time"
)

const (
	ScanQueued    = "Queued"
	ScanRunning   = "Running"
	ScanCompleted = "Completed"
	ScanFailed    = "Failed"
	ScanCanceled  = "Canceled"
	ScanPartial   = "Partial"
)

type ScanStatus string

type CancelScanModel struct {
	Status ScanStatus `json:"status"`
}

type ScanTaskResponseModel struct {
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
	Info      string `json:"info"`
}

type Config struct {
	Type  string                 `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Value map[string]interface{} `protobuf:"bytes,2,rep,name=value,proto3" json:"value,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

type ScanHandler struct {
	// representative branch
	Branch string `json:"branch,omitempty"`
	// representative repository url
	RepoURL   string `json:"repoUrl"`
	UploadURL string `json:"uploadUrl"`
	// Credentials
	Credentials GitCredentials `json:"credentials"`
}

type GitProjectHandler struct {
	RepoURL     string         `json:"repoUrl"`
	Branch      string         `json:"branch,omitempty"`
	Commit      string         `json:"commit,omitempty"`
	Tag         string         `json:"tag,omitempty"`
	Credentials GitCredentials `json:"credentials"`
}

type GitCredentials struct {
	// The user name required for accessing the repository
	Username string `json:"username,omitempty"`
	Type     string `json:"type"` // [apiKey|password|ssh|JWT]
	Value    string `json:"value,omitempty"`
}

type StatusInfo struct {
	Name      string
	Status    string
	Details   string `json:"details,omitempty"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

type ScanResponseModel struct {
	ID              string                    `json:"id"`
	Status          ScanStatus                `json:"status"`
	PositionInQueue *uint                     `json:"positionInQueue,omitempty"`
	StatusDetails   []StatusInfo              `json:"statusDetails,omitempty"`
	Branch          string                    `json:"branch"`
	CreatedAt       time.Time                 `json:"createdAt"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
	ProjectID       string                    `json:"projectId"`
	ProjectName     string                    `json:"projectName"`
	UserAgent       string                    `json:"userAgent"`
	Initiator       string                    `json:"initiator"`
	Tags            map[string]string         `json:"tags"`
	Metadata        ScanResponseModelMetadata `json:"metadata"`
	Engines         []string                  `json:"engines"`
	SourceType      string                    `json:"sourceType"`
	SourceOrigin    string                    `json:"sourceOrigin"`
	SastIncremental string                    `json:"sastIncremental"`
	Timeout         string                    `json:"timeout"`
}

type ScanResponseModelMetadata struct {
	Configs []Config `json:"configs"`
}

type ScansCollectionResponseModel struct {
	TotalCount         uint                `json:"totalCount"`
	FilteredTotalCount uint                `json:"filteredTotalCount"`
	Scans              []ScanResponseModel `json:"scans"`
}

type ScanProject struct {
	// An identifier for a project. A project is a way to group scans.
	// Project id can be empty, and if none provided, a project id and name will be created.
	// For example in git repositories, the created project name will be the repository URL and id will be a new uuid.
	ID   string            `json:"id"`
	Tags map[string]string `json:"tags"`
}

type Scan struct {
	Type    string            `json:"type"`    // [git|upload]
	Handler json.RawMessage   `json:"handler"` // One of [GitProjectHandler|ScanHandler]
	Project ScanProject       `json:"project,omitempty"`
	Config  []Config          `json:"config,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

type ScansWrapper interface {
	Create(model *Scan) (*ScanResponseModel, *ErrorModel, error)
	Get(params map[string]string) (*ScansCollectionResponseModel, *ErrorModel, error)
	GetByID(scanID string) (*ScanResponseModel, *ErrorModel, error)
	GetWorkflowByID(scanID string) ([]*ScanTaskResponseModel, *ErrorModel, error)
	Delete(scanID string) (*ErrorModel, error)
	Cancel(scanID string) (*ErrorModel, error)
	Tags() (map[string][]string, *ErrorModel, error)
}

type SastConfig struct {
	Incremental   string `json:"incremental,omitempty"`
	Filter        string `json:"filter,omitempty"`
	EngineVerbose string `json:"engineVerbose,omitempty"`
	LanguageMode  string `json:"languageMode,omitempty"`
	PresetName    string `json:"presetName,omitempty"`
	FastScanMode  string `json:"fastScanMode,omitempty"`
}

type KicsConfig struct {
	Filter    string `json:"filter,omitempty"`
	Platforms string `json:"platforms,omitempty"`
}

type ScaConfig struct {
	Filter                string `json:"filter,omitempty"`
	ExploitablePath       string `json:"ExploitablePath,omitempty"`
	LastSastScanTime      string `json:"LastSastScanTime,omitempty"`
	PrivatePackageVersion string `json:"privatePackageVersion,omitempty"`
}
type APISecConfig struct {
	SwaggerFilter string `json:"swaggerFilter,omitempty"`
}
