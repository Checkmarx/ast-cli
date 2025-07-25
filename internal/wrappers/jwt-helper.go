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

type JWTWrapper interface {
	GetAllowedEngines(featureFlagsWrapper FeatureFlagsWrapper) (allowedEngines map[string]bool, scsLicensingV2 bool, err error)
	IsAllowedEngine(engine string) (bool, error)
	ExtractTenantFromToken() (tenant string, err error)
}

func NewJwtWrapper() JWTWrapper {
	return &JWTStruct{}
}

func getEnabledEngines(scsLicensingV2 bool) (enabledEngines []string) {
	enabledEngines = []string{"sast", "sca", "api-security", "iac-security", "containers"}
	if scsLicensingV2 {
		enabledEngines = append(enabledEngines, commonParams.RepositoryHealthType, commonParams.SecretDetectionType)
	} else {
		enabledEngines = append(enabledEngines, commonParams.ScsType, commonParams.EnterpriseSecretsType)
	}
	return enabledEngines
}

func getDefaultEngines(scsLicensingV2 bool) (defaultEngines map[string]bool) {
	defaultEngines = map[string]bool{
		"sast":         true,
		"sca":          true,
		"api-security": true,
		"iac-security": true,
		"containers":   true,
	}
	if scsLicensingV2 {
		defaultEngines[commonParams.RepositoryHealthType] = true
		defaultEngines[commonParams.SecretDetectionType] = true
	} else {
		defaultEngines[commonParams.ScsType] = true
		defaultEngines[commonParams.EnterpriseSecretsType] = true
	}
	return defaultEngines
}

// GetAllowedEngines will return a map with user allowed engines
func (*JWTStruct) GetAllowedEngines(featureFlagsWrapper FeatureFlagsWrapper) (allowedEngines map[string]bool, scsLicensingV2 bool, err error) {
	scsLicensingV2Flag, _ := GetSpecificFeatureFlag(featureFlagsWrapper, ScsLicensingV2Enabled)
	scsLicensingV2 = scsLicensingV2Flag.Status
	flagResponse, _ := GetSpecificFeatureFlag(featureFlagsWrapper, PackageEnforcementEnabled)
	if flagResponse.Status {
		jwtStruct, err := getJwtStruct()
		if err != nil {
			return nil, scsLicensingV2, err
		}
		allowedEngines = prepareEngines(jwtStruct.AstLicense.LicenseData.AllowedEngines, scsLicensingV2)
		return allowedEngines, scsLicensingV2, nil
	}

	return getDefaultEngines(scsLicensingV2), scsLicensingV2, nil
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

func prepareEngines(engines []string, scsLicensingV2 bool) map[string]bool {
	m := make(map[string]bool)
	for _, value := range engines {
		engine := strings.Replace(strings.ToLower(value), strings.ToLower(commonParams.APISecurityLabel), commonParams.APISecurityType, 1)
		engine = strings.Replace(strings.ToLower(engine), commonParams.KicsType, commonParams.IacType, 1)
		if scsLicensingV2 {
			engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.RepositoryHealthLabel), commonParams.RepositoryHealthType, 1)
			engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.SecretDetectionLabel), commonParams.SecretDetectionType, 1)
		} else {
			engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.EnterpriseSecretsLabel), commonParams.EnterpriseSecretsType, 1)
		}

		// Current limitation, CxOne is including non-engines in the JWT
		enabledEngines := getEnabledEngines(scsLicensingV2)
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
