package mock

import "github.com/checkmarxDev/ast-cli/internal/wrappers"

type AuthMockWrapper struct{}

func (a *AuthMockWrapper) CreateOauth2Client(*wrappers.Oath2Client, string, string, string, string) (*wrappers.ErrorMsg, error) {
	return nil, nil
}

func (a *AuthMockWrapper) SetPath(_ string) {
}
