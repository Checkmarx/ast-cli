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

var defaultEngines = map[string]bool{
	"sast":         true,
	"sca":          true,
	"api-security": true,
	"iac-security": true,
	"microengines": true,
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
