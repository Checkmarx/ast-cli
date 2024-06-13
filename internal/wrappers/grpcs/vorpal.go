package grpcs

type VorpalWrapper interface {
	Scan(fileName, sourceCode string) (*ScanResult, error)
	HealthCheck() error
	ShutDown() error
}

type ScanResult struct {
	RequestID   string       `json:"request_id"`
	Status      bool         `json:"status"`
	Message     string       `json:"message"`
	ScanDetails []ScanDetail `json:"scan_details"`
	Error       *Error       `json:"error"`
}

type ScanDetail struct {
	RuleID          uint32 `json:"rule_id"`
	Language        string `json:"language"`
	RuleName        string `json:"rule_name"`
	Severity        string `json:"severity"`
	FileName        string `json:"file_name"`
	Line            uint32 `json:"line"`
	ProblematicLine string `json:"problematicLine"`
	Length          uint32 `json:"length"`
	Remediation     string `json:"remediationAdvise"`
	Description     string `json:"description"`
}

type Error struct {
	Code        ErrorCode `json:"code"`
	Description string    `json:"description"`
}

type ErrorCode int32

const (
	UnknownError   = 0
	InvalidRequest = 1
	InternalError  = 2
)
