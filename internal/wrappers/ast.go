package wrappers

import (
	"encoding/json"
	"fmt"
)

type ErrorModel struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

type WebError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type AstError struct {
	Code int
	Err  error
}

func (er *AstError) Error() string {
	return fmt.Sprintf("%v", er.Err)
}

func (er *AstError) Unwrap() error {
	return er.Err
}

func NewAstError(code int, err error) *AstError {
	return &AstError{
		Code: code,
		Err:  err,
	}
}

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
	TenantId          string        `json:"tenant_id"`
	TenantName        string        `json:"tenant_name"`
	EmailVerified     bool          `json:"email_verified"`
	Roles             []string      `json:"roles"`
	EulaAccepted      bool          `json:"eula-accepted"`
	Groups            []interface{} `json:"groups"`
	GroupsNames       []interface{} `json:"groupsNames"`
	CbUrl             string        `json:"cb-url"`
	PreferredUsername string        `json:"preferred_username"`
	GivenName         string        `json:"given_name"`
	AstBaseUrl        string        `json:"ast-base-url"`
	SfId              string        `json:"sf-id"`
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
			CodeBashingUrl                  string   `json:"codeBashingUrl"`
			ExpirationDate                  int64    `json:"expirationDate"`
			UnlimitedProjects               bool     `json:"unlimitedProjects"`
			CodeBashingEnabled              bool     `json:"codeBashingEnabled"`
			MaxConcurrentScans              int      `json:"maxConcurrentScans"`
			CodeBashingUsersCount           int      `json:"codeBashingUsersCount"`
			CustomMaxConcurrentScansEnabled bool     `json:"customMaxConcurrentScansEnabled"`
		} `json:"LicenseData"`
		PackageName string `json:"PackageName"`
	} `json:"ast-license"`
	ServiceUsersEnabled bool   `json:"service_users_enabled"`
	FamilyName          string `json:"family_name"`
	Email               string `json:"email"`
	Tenant              string `json:"tenant"`
}
