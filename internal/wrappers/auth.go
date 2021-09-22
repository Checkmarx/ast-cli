package wrappers

type Oath2Client struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
	Secret      string   `json:"secret"`
}

type ErrorMsg struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

type AuthWrapper interface {
	CreateOauth2Client(client *Oath2Client, username, password, adminClientID, adminClientSecret string) (*ErrorMsg, error)
	SetPath(path string)
	ValidateLogin() error
}
