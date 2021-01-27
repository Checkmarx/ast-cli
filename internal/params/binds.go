package params

var EnvVarsBinds = []struct {
	Key     string
	Env     string
	Default string
}{
	{BaseURIKey, BaseURIEnv, "http://ast.cxflow-ci.com"},
	{ScansPathKey, ScansPathEnv, "api/scans"},
	{ProjectsPathKey, ProjectsPathEnv, "api/projects"},
	{ResultsPathKey, ResultsPathEnv, "api/results"},
	{BflPathKey, BflPathEnv, "api/bfl"},
	{UploadsPathKey, UploadsPathEnv, "api/uploads"},
	{SastRmPathKey, SastRmPathEnv, "api/sast-rm"},
	{AstWebAppHealthCheckPathKey, AstWebAppHealthCheckPathEnv, "#/projects"},
	{AstKeycloakWebAppHealthCheckPathKey, AstKeycloakWebAppHealthCheckPathEnv, "auth"},
	{HealthcheckPathKey, HealthcheckPathEnv, "api/healthcheck"},
	{HealthcheckDBPathKey, HealthcheckDBPathEnv, "database"},
	{HealthcheckMessageQueuePathKey, HealthcheckMessageQueuePathEnv, "message-queue"},
	{HealthcheckObjectStorePathKey, HealthcheckObjectStorePathEnv, "object-store"},
	{HealthcheckInMemoryDBPathKey, HealthcheckInMemoryDBPathEnv, "in-memory-db"},
	{HealthcheckLoggingPathKey, HealthcheckLoggingPathEnv, "logging"},
	{HealthcheckScanFlowPathKey, HealthcheckScanFlowPathEnv, "scan-flow"},
	{HealthcheckSastEnginesPathKey, HealthcheckSastEnginesPathEnv, "sast-engines"},
	{QueriesPathKey, QueriesPathEnv, "api/queries"},
	{QueriesClonePathKey, QueriesCLonePathEnv, "clone"},
	{CreateOath2ClientPathKey, CreateOath2ClientPathEnv, "auth/realms/organization/pip/clients"},
	{SastScanIncPathKey, SastScanIncPathEnv, "api/sast-scan-inc"},
	{SastScanIncEngineLogPathKey, SastScanIncEngineLogPathEnv, "%s/engine-log"},
	{SastScanIncMetricsPathKey, SastScanIncMetricsPathEnv, "%s/metrics"},
	{LogsPathKey, LogsPathEnv, "api/logs"},
	{AccessKeyIDConfigKey, AccessKeyIDEnv, "ast-plugins-3d1b8499-b3cf-43cf-8425-0bb212ca44d3"},
	{AccessKeySecretConfigKey, AccessKeySecretEnv, "710a92f-d103-4bc5-89ce-6d138f976b9e"},
	{AstAuthenticationPathConfigKey, AstAuthenticationPathEnv, "auth/realms/organization/protocol/openid-connect/token"},
	{AstRoleKey, AstRoleEnv, ScaAgent},
	{CredentialsFilePathKey, CredentialsFilePathEnv, "credentials.json"},
	{TokenExpirySecondsKey, TokenExpirySecondsEnv, "300"},
}
