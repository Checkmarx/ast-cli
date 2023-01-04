package mock

import (
	"strings"
)

type JWTMockWrapper struct{}

// GetAllowedEngines mock for tests
func (*JWTMockWrapper) GetAllowedEngines() (err error, allowedEngines map[string]bool) {
	allowedEngines = make(map[string]bool)
	engines := []string{"sast", "kics", "sca", "api-security"}
	for _, value := range engines {
		allowedEngines[strings.ToLower(value)] = true
	}
	return nil, allowedEngines
}
