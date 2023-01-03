package mock

import (
	"strings"
)

type JWTMockWrapper struct{}

func (a *JWTMockWrapper) GetAllowedEngines() (error, map[string]bool) {
	m := make(map[string]bool)
	engines := []string{"sast", "kics", "sca", "api-security"}
	for _, value := range engines {
		m[strings.ToLower(value)] = true
	}
	return nil, m
}
