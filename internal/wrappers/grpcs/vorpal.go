package grpcs

type VorpalWrapper interface {
	Scan(fileName, sourceCode string) (*ScanResult, error)
	HealthCheck() error
	ShutDown() error
	GetPort() int
	ConfigurePort(port int)
}

type ScanResult struct {
	RequestID   string       `json:"request_id,omitempty"`
	Status      bool         `json:"status,omitempty"`
	Message     string       `json:"message,omitempty"`
	ScanDetails []ScanDetail `json:"scan_details,omitempty"`
	Error       *Error       `json:"error,omitempty"`
}

type ScanDetail struct {
	RuleID          uint32 `json:"rule_id,omitempty"`
	Language        string `json:"language,omitempty"`
	RuleName        string `json:"rule_name,omitempty"`
	Severity        string `json:"severity,omitempty"`
	FileName        string `json:"file_name,omitempty"`
	Line            uint32 `json:"line,omitempty"`
	ProblematicLine string `json:"problematicLine,omitempty"`
	Length          uint32 `json:"length,omitempty"`
	Remediation     string `json:"remediationAdvise,omitempty"`
	Description     string `json:"description,omitempty"`
}

type Error struct {
	Code        ErrorCode `json:"code,omitempty"`
	Description string    `json:"description,omitempty"`
}

type ErrorCode int32

const (
	UnknownError   = 0
	InvalidRequest = 1
	InternalError  = 2
)
