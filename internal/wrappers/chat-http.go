package wrappers

import (
	gptWrapperMaskedSecret "github.com/checkmarxDev/gpt-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/checkmarxDev/gpt-wrapper/pkg/message"
	gptWrapper "github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatHTTPWrapper struct {
}

func (c ChatHTTPWrapper) MaskSecrets(w gptWrapper.StatefulWrapper, fileContent string) (*gptWrapperMaskedSecret.MaskedEntry, error) {
	return w.MaskSecrets(fileContent)
}

func (c ChatHTTPWrapper) Call(w gptWrapper.StatefulWrapper, id uuid.UUID, messages []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return w.Call(id, messages)
}

func NewChatWrapper() ChatWrapper {
	return ChatHTTPWrapper{}
}
