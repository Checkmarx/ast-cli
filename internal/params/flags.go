package params

const (
	IDQueryParam           = "id"
	IDsQueryParam          = "ids"
	IDRegexQueryParam      = "id-regex"
	LimitQueryParam        = "limit"
	OffsetQueryParam       = "offset"
	ScanIDQueryParam       = "scan-id"
	ScanIDsQueryParam      = "scan-ids"
	TagsKeyQueryParam      = "tags-keys"
	TagsValueQueryParam    = "tags-values"
	StatusesQueryParam     = "statuses"
	StatusQueryParam       = "status"
	ProjectIDQueryParam    = "project-id"
	FromDateQueryParam     = "from-date"
	ToDateQueryParam       = "to-date"
	SeverityQueryParam     = "severity"
	GroupQueryParam        = "group"
	QueryQueryParam        = "query"
	NodeIDsQueryParam      = "node-ids"
	IncludeNodesQueryParam = "include-nodes"
	SortQueryParam         = "sort"
	Profile                = "default"
	BaseURI                = ""
	AgentFlag              = "ASTCLI"
	BaseIAMURI             = ""
	Tenant                 = ""
	Branch                 = ""
)

// ScaAgent AST Role
const ScaAgent = "SCA_AGENT"

var (
	Version = "dev"
)
