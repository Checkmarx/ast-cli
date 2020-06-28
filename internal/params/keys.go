package params

import "strings"

var (
	BaseURIKey                          = strings.ToLower(BaseURIEnv)
	ScansPathKey                        = strings.ToLower(ScansPathEnv)
	ProjectsPathKey                     = strings.ToLower(ProjectsPathEnv)
	ResultsPathKey                      = strings.ToLower(ResultsPathEnv)
	BflPathKey                          = strings.ToLower(BflPathEnv)
	UploadsPathKey                      = strings.ToLower(UploadsPathEnv)
	AccessKeyIDConfigKey                = strings.ToLower(AccessKeyIDEnv)
	AccessKeySecretConfigKey            = strings.ToLower(AccessKeySecretEnv)
	AstAuthenticationURIConfigKey       = strings.ToLower(AstAuthenticationURIEnv)
	CredentialsFilePathKey              = strings.ToLower(CredentialsFilePathEnv)
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
	HealthcheckGetAstRolePathKey        = strings.ToLower(HealthcheckGetAstRolePathEnv)
	ScanHealthCheckSourcePathKey        = strings.ToLower(ScanHealthCheckSourcePathEnv)
	ScanHealthCheckTimeoutSecsKey       = strings.ToLower(ScanHealthCheckTimeoutSecsEnv)
)
