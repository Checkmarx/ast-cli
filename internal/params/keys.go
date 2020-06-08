package params

import "strings"

var (
	AstURIKey                     = strings.ToLower(AstURIEnv)
	ScansPathKey                  = strings.ToLower(ScansPathEnv)
	ProjectsPathKey               = strings.ToLower(ProjectsPathEnv)
	ResultsPathKey                = strings.ToLower(ResultsPathEnv)
	BflPathKey                    = strings.ToLower(BflPathEnv)
	UploadsPathKey                = strings.ToLower(UploadsPathEnv)
	AccessKeyIDConfigKey          = strings.ToLower(AccessKeyIDEnv)
	AccessKeySecretConfigKey      = strings.ToLower(AccessKeySecretEnv)
	AstAuthenticationURIConfigKey = strings.ToLower(AstAuthenticationURIEnv)
	CredentialsFilePathKey        = strings.ToLower(CredentialsFilePathEnv)
	TokenExpirySecondsKey         = strings.ToLower(TokenExpirySecondsEnv)
	AstRoleKey                    = strings.ToLower(AstRoleEnv)
	AstWebAppHealthCheckPathKey   = strings.ToLower(AstWebAppHealthCheckPathEnv)
	AstHealthcheckURIKey          = strings.ToLower(AstHealthcheckURIEnv)
	AstHealthcheckDBPathKey       = strings.ToLower(AstHealthcheckDBPathEnv)
)
