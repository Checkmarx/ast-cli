package wrappers

import (
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

const (
	astLicenseMapKey     = "ast-license"
	astLicenseDataMapKey = "LicenseData"
	astAllowedEnginesKey = "allowedEngines"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
	LicenseData struct {
		AllowedEngines []string `json:"allowedEngines"`
	} `json:"LicenseData"`
	//jwt.StandardClaims
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

func extractFromTokenToJwtStruct(accessToken string) (*JWTStruct, error) {
	value := &JWTStruct{}
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims[astLicenseMapKey] != nil {
		astLicenseClaim := claims[astLicenseMapKey].(map[string]interface{})
		licenseData := astLicenseClaim[astLicenseDataMapKey].(map[string]interface{})
		value.LicenseData.AllowedEngines = utils.ToStringArray(licenseData[astAllowedEnginesKey])
	}
	return value, nil
}
