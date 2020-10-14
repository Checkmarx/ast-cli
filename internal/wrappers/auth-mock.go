package wrappers

type AuthMockWrapper struct{}

func (a *AuthMockWrapper) CreateOauth2Client(*Oath2Client, string, string, string, string) (*ErrorMsg, error) {
	return nil, nil
}
