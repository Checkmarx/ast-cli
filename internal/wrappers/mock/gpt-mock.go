package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type KicsHTTPGptWrapper struct {
}

func (g *KicsHTTPGptWrapper) SendToChatGPT(conv *wrappers.Conversation) (string, error) {
	return "", nil
}
