package wrappers

import (
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
	Exp            int      `json:"exp"`
	Iat            int      `json:"iat"`
	Jti            string   `json:"jti"`
	Iss            string   `json:"iss"`
	Aud            []string `json:"aud"`
	Sub            string   `json:"sub"`
	Typ            string   `json:"typ"`
	Azp            string   `json:"azp"`
	SessionState   string   `json:"session_state"`
	AllowedOrigins []string `json:"allowed-origins"`
	ResourceAccess struct {
		RealmManagement struct {
			Roles []string `json:"roles"`
		} `json:"realm-management"`
		Account struct {
			Roles []string `json:"roles"`
		} `json:"account"`
	} `json:"resource_access"`
	Scope             string        `json:"scope"`
	Sid               string        `json:"sid"`
	TenantID          string        `json:"tenant_id"`
	TenantName        string        `json:"tenant_name"`
	EmailVerified     bool          `json:"email_verified"`
	Roles             []string      `json:"roles"`
	EulaAccepted      bool          `json:"eula-accepted"`
	Groups            []interface{} `json:"groups"`
	GroupsNames       []interface{} `json:"groupsNames"`
	CbURL             string        `json:"cb-url"`
	PreferredUsername string        `json:"preferred_username"`
	GivenName         string        `json:"given_name"`
	AstBaseURL        string        `json:"ast-base-url"`
	SfID              string        `json:"sf-id"`
	RolesAst          []string      `json:"roles_ast"`
	Name              string        `json:"name"`
	TenantType        string        `json:"tenant-type"`
	AstLicense        struct {
		ID          int    `json:"ID"`
		TenantID    string `json:"TenantID"`
		IsActive    bool   `json:"IsActive"`
		PackageID   int    `json:"PackageID"`
		LicenseData struct {
			Features                        []string `json:"features"`
			Services                        []string `json:"services"`
			UsersCount                      int      `json:"usersCount"`
			ServiceType                     string   `json:"serviceType"`
			ActivationDate                  int64    `json:"activationDate"`
			AllowedEngines                  []string `json:"allowedEngines"`
			CodeBashingURL                  string   `json:"codeBashingUrl"`
			ExpirationDate                  int64    `json:"expirationDate"`
			UnlimitedProjects               bool     `json:"unlimitedProjects"`
			CodeBashingEnabled              bool     `json:"codeBashingEnabled"`
			MaxConcurrentScans              int      `json:"maxConcurrentScans"`
			CodeBashingUsersCount           int      `json:"codeBashingUsersCount"`
			CustomMaxConcurrentScansEnabled bool     `json:"customMaxConcurrentScansEnabled"`
		} `json:"LicenseData"`
		PackageName string `json:"PackageName"`
	} `json:"ast-license"`
	jwt.StandardClaims
}

type JWTWrapper interface {
	GetAllowedEngines() (allowedEngines map[string]bool, err error)
}

func NewJwtWrapper() JWTWrapper {
	return &JWTStruct{}
}

// GetAllowedEngines will return a map with user allowed engines
func (*JWTStruct) GetAllowedEngines() (allowedEngines map[string]bool, err error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	jwtStruct, _ := extractFromTokenToJwtStruct(accessToken)
	allowedEngines = fillBooleanMap(jwtStruct.AstLicense.LicenseData.AllowedEngines)

	return allowedEngines, nil
}

func fillBooleanMap(engines []string) map[string]bool {
	m := make(map[string]bool)
	for _, value := range engines {
		engine := strings.Replace(strings.ToLower(value), strings.ToLower(commonParams.APISecurityLabel), commonParams.APISecurityType, 1)
		m[strings.ToLower(engine)] = true
	}
	return m
}

// extractFromTokenToJwtStruct used in scan validation
func extractFromTokenToJwtStruct(accessToken string) (*JWTStruct, error) {
	claims := JWTStruct{}
	_, _, err := new(jwt.Parser).ParseUnverified(accessToken, &claims)
	if err != nil {
		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
	}
	return &claims, nil
}
