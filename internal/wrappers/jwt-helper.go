package wrappers

import (
	"strings"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
	Tenant     string `json:"tenant_name"`
	AstLicense struct {
		LicenseData struct {
			AllowedEngines []string `json:"allowedEngines"`
		} `json:"LicenseData"`
	} `json:"ast-license"`
	jwt.RegisteredClaims // Embedding the standard claims
}

var enabledEngines = []string{"sast", "sca", "api-security", "iac-security", "scs", "containers", "enterprise-secrets"}

var defaultEngines = map[string]bool{
	"sast":         true,
	"sca":          true,
	"api-security": true,
	"iac-security": true,
	"containers":   true,
	"scs":          true,
}

type JWTWrapper interface {
	GetAllowedEngines(featureFlagsWrapper FeatureFlagsWrapper) (allowedEngines map[string]bool, err error)
	IsAllowedEngine(engine string) (bool, error)
	ExtractTenantFromToken() (tenant string, err error)
}

func NewJwtWrapper() JWTWrapper {
	return &JWTStruct{}
}

// GetAllowedEngines will return a map with user allowed engines
func (*JWTStruct) GetAllowedEngines(featureFlagsWrapper FeatureFlagsWrapper) (allowedEngines map[string]bool, err error) {
	flagResponse, _ := GetSpecificFeatureFlag(featureFlagsWrapper, PackageEnforcementEnabled)
	if flagResponse.Status {
		jwtStruct, err := getJwtStruct()
		if err != nil {
			return nil, err
		}
		allowedEngines = prepareEngines(jwtStruct.AstLicense.LicenseData.AllowedEngines)
		return allowedEngines, nil
	}

	return defaultEngines, nil
}

func getJwtStruct() (*JWTStruct, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	return extractFromTokenToJwtStruct(accessToken)
}

// IsAllowedEngine will return if the engine is allowed in the user license
func (*JWTStruct) IsAllowedEngine(engine string) (bool, error) {
	jwtStruct, err := getJwtStruct()
	if err != nil {
		return false, err
	}

	for _, allowedEngine := range jwtStruct.AstLicense.LicenseData.AllowedEngines {
		if strings.EqualFold(allowedEngine, engine) {
			return true, nil
		}
	}
	return false, nil
}

func prepareEngines(engines []string) map[string]bool {
	m := make(map[string]bool)
	for _, value := range engines {
		engine := strings.Replace(strings.ToLower(value), strings.ToLower(commonParams.APISecurityLabel), commonParams.APISecurityType, 1)
		engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.EnterpriseSecretsLabel), commonParams.EnterpriseSecretsType, 1)
		engine = strings.Replace(strings.ToLower(engine), commonParams.KicsType, commonParams.IacType, 1)

		// Current limitation, CxOne is including non-engines in the JWT
		if utils.Contains(enabledEngines, strings.ToLower(engine)) {
			m[strings.ToLower(engine)] = true
		}
	}
	return m
}

func extractFromTokenToJwtStruct(accessToken string) (*JWTStruct, error) {
	// Create a new Parser instance
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(accessToken, &JWTStruct{})
	if err != nil {
		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
	}

	claims, ok := token.Claims.(*JWTStruct)
	if !ok {
		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
	}

	return claims, nil
}

func (*JWTStruct) ExtractTenantFromToken() (tenant string, err error) {
	jwtStruct, err := getJwtStruct()
	if err != nil {
		return "", err
	}
	return jwtStruct.Tenant, nil
}
