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
	BaseURI                = "127.0.0.1:80"
	AgentFlag              = "ASTCLI"
	BaseIAMURI             = ""
	AstToken               = ""
	Tenant                 = ""
	Branch                 = ""
)

const (
	// Roles
	ScaAgent     = "SCA_AGENT"
	SastEngine   = "SAST_ENGINE"
	SastALlInOne = "SAST_ALL_IN_ONE"
)

const (
	Version = "2.0.0_RC12"
)
