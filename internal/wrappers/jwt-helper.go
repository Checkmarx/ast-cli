package wrappers

import (
	"os/user"
	"strconv"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
	Tenant     string `json:"tenant_name"`
	AstLicense struct {
		LicenseData struct {
			AllowedEngines []string `json:"allowedEngines"`
			DastEnabled    bool     `json:"dastEnabled"`
		} `json:"LicenseData"`
	} `json:"ast-license"`
	ASTRoles             []string `json:"roles_ast"`
	jwt.RegisteredClaims          // Embedding the standard claims
}

type JWTWrapper interface {
	GetAllowedEngines(featureFlagsWrapper FeatureFlagsWrapper) (allowedEngines map[string]bool, err error)
	GetLicenseDetails() (licenseDetails map[string]string, err error)
	IsAllowedEngine(engine string) (bool, error)
	ExtractTenantFromToken() (tenant string, err error)
	CheckPermissionByAccessToken(requiredPermission string) (permission bool, err error)
}

func NewJwtWrapper() JWTWrapper {
	return &JWTStruct{}
}

func getEnabledEngines() []string {
	return []string{"sast", "sca", "api-security", "iac-security", "containers", "aisc",
		commonParams.RepositoryHealthType, commonParams.SecretDetectionType}
}

func getDefaultEngines() map[string]bool {
	return map[string]bool{
		"sast":                            true,
		"sca":                             true,
		"api-security":                    true,
		"iac-security":                    true,
		"containers":                      true,
		"aisc":                            true,
		commonParams.RepositoryHealthType: true,
		commonParams.SecretDetectionType:  true,
	}
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

	return getDefaultEngines(), nil
}

func (*JWTStruct) GetLicenseDetails() (licenseDetails map[string]string, err error) {
	jwtStruct, err := getJwtStruct()
	if err != nil {
		return nil, err
	}
	return buildLicenseDetailsFromJWT(jwtStruct), nil
}

func buildLicenseDetailsFromJWT(jwtStruct *JWTStruct) map[string]string {
	licenseDetails := make(map[string]string)

	assistEnabled := containsIgnoreCase(jwtStruct.AstLicense.LicenseData.AllowedEngines, commonParams.CheckmarxOneAssistType) ||
		containsIgnoreCase(jwtStruct.AstLicense.LicenseData.AllowedEngines, commonParams.AIProtectionType)
	devAssistEnabled := containsIgnoreCase(jwtStruct.AstLicense.LicenseData.AllowedEngines, commonParams.CheckmarxDevAssistType)

	licenseDetails[commonParams.CxOneAssistEnabledKey] = strconv.FormatBool(assistEnabled)
	licenseDetails[commonParams.CxDevAssistEnabledKey] = strconv.FormatBool(devAssistEnabled)
	licenseDetails[commonParams.DastEnabledKey] = strconv.FormatBool(jwtStruct.AstLicense.LicenseData.DastEnabled)

	return licenseDetails
}

// containsIgnoreCase returns true if target exists in arr using case-insensitive comparison
func containsIgnoreCase(arr []string, target string) bool {
	for _, s := range arr {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
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
		engine = strings.Replace(strings.ToLower(engine), commonParams.KicsType, commonParams.IacType, 1)
		engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.RepositoryHealthLabel), commonParams.RepositoryHealthType, 1)
		engine = strings.Replace(strings.ToLower(engine), strings.ToLower(commonParams.SecretDetectionLabel), commonParams.SecretDetectionType, 1)

		// Current limitation, CxOne is including non-engines in the JWT
		enabledEngines := getEnabledEngines()
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
func (*JWTStruct) CheckPermissionByAccessToken(requiredPermission string) (hasPermission bool, err error) {
	jwtStruct, err := getJwtStruct()
	if err != nil {
		return false, err
	}
	permission := false
	for _, role := range jwtStruct.ASTRoles {
		if role == requiredPermission {
			permission = true
			break
		}
	}
	return permission, nil
}

func GetUniqueID() string {
	var uniqueID string
	// Check License first
	jwtWrapper := NewJwtWrapper()
	devAssistAllowed, err := jwtWrapper.IsAllowedEngine(commonParams.CheckmarxDevAssistType)
	if err != nil {
		logger.PrintIfVerbose("Failed to check engine allowance: " + err.Error())
		return ""
	}

	oneAssistAllowed, err := jwtWrapper.IsAllowedEngine(commonParams.CheckmarxOneAssistType)
	if err != nil {
		logger.PrintIfVerbose("Failed to check engine allowance: " + err.Error())
		return ""
	}

	if !devAssistAllowed && !oneAssistAllowed {
		return ""
	}

	// Check if unique id is already set
	uniqueID = viper.GetString(commonParams.UniqueIDConfigKey)
	if uniqueID != "" {
		return uniqueID
	}

	// Generate new unique id
	currentUser, err := user.Current()
	if err != nil {
		logger.PrintIfVerbose("Failed to get user: " + err.Error())
		return ""
	}
	username := currentUser.Username
	username = strings.TrimSpace(username)
	if strings.Contains(username, "\\") {
		username = strings.Split(username, "\\")[1]
	}
	uniqueID = uuid.New().String() + "_" + username
	viper.Set(commonParams.UniqueIDConfigKey, uniqueID)
	configFilePath, _ := configuration.GetConfigFilePath()
	err = configuration.SafeWriteSingleConfigKeyString(configFilePath, commonParams.UniqueIDConfigKey, uniqueID)
	if err != nil {
		logger.PrintIfVerbose("Failed to write config: " + err.Error())
		return ""
	}
	return uniqueID
}
