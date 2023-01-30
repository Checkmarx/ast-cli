package wrappers

import (
	"encoding/json"
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
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
	jwt.StandardClaims
}

var enabledEngines = []string{"sast", "sca", "api-security", "iac-security"}

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
	jwtStruct, err := extractFromTokenToJwtStruct(accessToken)
	if err != nil {
		return nil, err
	}
	allowedEngines = prepareEngines(jwtStruct.LicenseData.AllowedEngines)
	return allowedEngines, nil
}

func prepareEngines(engines []string) map[string]bool {
	m := make(map[string]bool)
	for _, value := range engines {
		engine := strings.Replace(strings.ToLower(value), strings.ToLower(commonParams.APISecurityLabel), commonParams.APISecurityType, 1)
		engine = strings.Replace(strings.ToLower(engine), commonParams.KicsType, commonParams.IacType, 1)

		// Current limitation, CxOne is including non-engines in the JWT
		if utils.Contains(enabledEngines, strings.ToLower(engine)) {
			m[strings.ToLower(engine)] = true
		}
	}
	return m
}

// extractFromTokenToJwtStruct used in scan validation
func extractFromTokenToJwtStruct(accessToken string) (*JWTStruct, error) {
	value := &JWTStruct{}
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims["ast-license"] != nil {
		astLicenseClaim := claims["ast-license"]
		valueBytes, _ := json.Marshal(astLicenseClaim)
		err = json.Unmarshal(valueBytes, &value)
		if err != nil {
			return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
		}
	}
	if err != nil {
		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
	}
	return value, nil
}
