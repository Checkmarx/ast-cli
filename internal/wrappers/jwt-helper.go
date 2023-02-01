package wrappers

import (
	"github.com/golang-jwt/jwt"
)

// JWTStruct model used to get all jwt fields
type JWTStruct struct {
	AstLicense struct {
		LicenseData struct {
			AllowedEngines []string `json:"allowedEngines"`
		} `json:"LicenseData"`
	} `json:"ast-license"`
	jwt.Claims
}

var enabledEngines = []string{"sast", "sca", "api-security", "iac-security"}
var defaultEngines = map[string]bool{
	"sast":         true,
	"sca":          true,
	"api-security": true,
	"iac-security": true,
}

type JWTWrapper interface {
	GetAllowedEngines() (allowedEngines map[string]bool, err error)
}

func NewJwtWrapper() JWTWrapper {
	return &JWTStruct{}
}

func (*JWTStruct) GetAllowedEngines() (allowedEngines map[string]bool, err error) {
	return defaultEngines, nil
}

//// GetAllowedEngines will return a map with user allowed engines
//func (*JWTStruct) GetAllowedEngines() (allowedEngines map[string]bool, err error) {
//	accessToken, err := GetAccessToken()
//	if err != nil {
//		return nil, err
//	}
//	jwtStruct, err := extractFromTokenToJwtStruct(accessToken)
//	if err != nil {
//		return nil, err
//	}
//	_ = prepareEngines(jwtStruct.AstLicense.LicenseData.AllowedEngines)
//	return allowedEngines, nil
//}
//
//func prepareEngines(engines []string) map[string]bool {
//	m := make(map[string]bool)
//	for _, value := range engines {
//		engine := strings.Replace(strings.ToLower(value), strings.ToLower(commonParams.APISecurityLabel), commonParams.APISecurityType, 1)
//		engine = strings.Replace(strings.ToLower(engine), commonParams.KicsType, commonParams.IacType, 1)
//
//		// Current limitation, CxOne is including non-engines in the JWT
//		if utils.Contains(enabledEngines, strings.ToLower(engine)) {
//			m[strings.ToLower(engine)] = true
//		}
//	}
//	return m
//}
//
//func extractFromTokenToJwtStruct(accessToken string) (*JWTStruct, error) {
//	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, &JWTStruct{})
//	if err != nil {
//		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
//	}
//
//	claims, ok := token.Claims.(*JWTStruct)
//	if !ok {
//		return nil, errors.Errorf(APIKeyDecodeErrorFormat, err)
//	}
//
//	return claims, nil
//}
