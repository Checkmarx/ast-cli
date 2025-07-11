package params

import "strings"

var (
	CustomStatesAPIPathKey              = strings.ToLower(CustomStatesAPIPathEnv)
	TenantKey                           = strings.ToLower(TenantEnv)
	BranchKey                           = strings.ToLower(BranchEnv)
	BaseURIKey                          = strings.ToLower(BaseURIEnv)
	ProxyKey                            = strings.ToLower(ProxyEnv)
	ProxyTypeKey                        = strings.ToLower(ProxyTypeEnv)
	ProxyDomainKey                      = strings.ToLower(ProxyDomainEnv)
	BaseAuthURIKey                      = strings.ToLower(BaseAuthURIEnv)
	ClientTimeoutKey                    = strings.ToLower(ClientTimeoutEnv)
	AstAPIKey                           = strings.ToLower(AstAPIKeyEnv)
	ScansPathKey                        = strings.ToLower(ScansPathEnv)
	GroupsPathKey                       = strings.ToLower(GroupsPathEnv)
	AgentNameKey                        = strings.ToLower(AgentNameEnv)
	OriginKey                           = strings.ToLower(OriginEnv)
	IgnoreProxyKey                      = strings.ToLower(IgnoreProxyEnv)
	CodeBashingPathKey                  = strings.ToLower(CodeBashingPathEnv)
	ProjectsPathKey                     = strings.ToLower(ProjectsPathEnv)
	ApplicationsPathKey                 = strings.ToLower(ApplicationsPathEnv)
	ResultsPathKey                      = strings.ToLower(ResultsPathEnv)
	ScanSummaryPathKey                  = strings.ToLower(ScanSummaryPathEnv)
	RisksOverviewPathKey                = strings.ToLower(RisksOverviewPathEnv)
	ScsScanOverviewPathKey              = strings.ToLower(ScsScanOverviewPathEnv)
	SastResultsPathKey                  = strings.ToLower(SastResultsPathEnv)
	KicsResultsPathKey                  = strings.ToLower(KicsResultsPathEnv)
	BflPathKey                          = strings.ToLower(BflPathEnv)
	PRDecorationGithubPathKey           = strings.ToLower(PRDecorationGithubPathEnv)
	PRDecorationGitlabPathKey           = strings.ToLower(PRDecorationGitlabPathEnv)
	PRDecorationBitbucketCloudPathKey   = strings.ToLower(PRDecorationBitbucketCloudPathEnv)
	PRDecorationBitbucketServerPathKey  = strings.ToLower(PRDecorationBitbucketServerPathEnv)
	PRDecorationAzurePathKey            = strings.ToLower(PRDecorationAzurePathEnv)
	UploadsPathKey                      = strings.ToLower(UploadsPathEnv)
	SastRmPathKey                       = strings.ToLower(SastRmPathEnv)
	AccessKeyIDConfigKey                = strings.ToLower(AccessKeyIDEnv)
	AccessKeySecretConfigKey            = strings.ToLower(AccessKeySecretEnv)
	TokenExpirySecondsKey               = strings.ToLower(TokenExpirySecondsEnv)
	AstRoleKey                          = strings.ToLower(AstRoleEnv)
	AstWebAppHealthCheckPathKey         = strings.ToLower(AstWebAppHealthCheckPathEnv)
	AstKeycloakWebAppHealthCheckPathKey = strings.ToLower(AstKeycloakWebAppHealthCheckPathEnv)
	HealthcheckPathKey                  = strings.ToLower(HealthcheckPathEnv)
	HealthcheckDBPathKey                = strings.ToLower(HealthcheckDBPathEnv)
	HealthcheckMessageQueuePathKey      = strings.ToLower(HealthcheckMessageQueuePathEnv)
	HealthcheckObjectStorePathKey       = strings.ToLower(HealthcheckObjectStorePathEnv)
	HealthcheckInMemoryDBPathKey        = strings.ToLower(HealthcheckInMemoryDBPathEnv)
	HealthcheckLoggingPathKey           = strings.ToLower(HealthcheckDBPathEnv)
	HealthcheckScanFlowPathKey          = strings.ToLower(HealthcheckScanFlowPathEnv)
	HealthcheckSastEnginesPathKey       = strings.ToLower(HealthcheckSastEnginesPathEnv)
	QueriesPathKey                      = strings.ToLower(QueriesPathEnv)
	QueriesClonePathKey                 = strings.ToLower(QueriesCLonePathEnv)
	CreateOath2ClientPathKey            = strings.ToLower(CreateOath2ClientPathEnv)
	SastMetadataPathKey                 = strings.ToLower(SastScanIncPathEnv)
	SastMetadataMetricsPathKey          = strings.ToLower(SastScanIncMetricsPathEnv)
	LogsPathKey                         = strings.ToLower(LogsPathEnv)
	LogsEngineLogPathKey                = strings.ToLower(LogsEngineLogPathEnv)
	SastResultsPredicatesPathKey        = strings.ToLower(SastResultsPredicatesPathEnv)
	KicsResultsPredicatesPathKey        = strings.ToLower(KicsResultsPredicatesPathEnv)
	ScsResultsReadPredicatesPathKey     = strings.ToLower(ScsResultsReadPredicatesPathEnv)
	ScsResultsWritePredicatesPathKey    = strings.ToLower(ScsResultsWritePredicatesPathEnv)
	DescriptionsPathKey                 = strings.ToLower(DescriptionsPathEnv)
	TenantConfigurationPathKey          = strings.ToLower(TenantConfigurationPathEnv)
	ResultsPdfReportPathKey             = strings.ToLower(ResultsPdfReportPathEnv)
	ResultsJSONReportPathKey            = strings.ToLower(ResultsJSONReportPathEnv)
	ExportPathKey                       = strings.ToLower(ExportPathEnv)
	FeatureFlagsKey                     = strings.ToLower(FeatureFlagsEnv)
	PolicyEvaluationPathKey             = strings.ToLower(PolicyEvaluationPathEnv)
	AccessManagementPathKey             = strings.ToLower(AccessManagementPathEnv)
	ByorPathKey                         = strings.ToLower(ByorPathEnv)
	AiProxyAzureAiRouteKey              = strings.ToLower(AiProxyAzureAiRouteEnv)
	AiProxyCheckmarxAiRouteKey          = strings.ToLower(AiProxyCheckmarxAiRouteEnv)
	ASCAPortKey                         = strings.ToLower(ASCAPortEnv)
	ScsRepoTokenKey                     = strings.ToLower(ScsRepoTokenEnv)
	RiskManagementPathKey               = strings.ToLower(RiskManagementPathEnv)
	ConfigFilePathKey                   = strings.ToLower(ConfigFilePathEnv)
	RealtimeScannerPathKey              = strings.ToLower(RealtimeScannerPathEnv)
)
