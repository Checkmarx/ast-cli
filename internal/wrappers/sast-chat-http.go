package wrappers

import (
	gptWrapperMaskedSecret "github.com/checkmarxDev/gpt-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/checkmarxDev/gpt-wrapper/pkg/message"
	gptWrapper "github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type SastChatHTTPWrapper struct {
}

func (c SastChatHTTPWrapper) MaskSecrets(w gptWrapper.StatefulWrapper, fileContent string) (*gptWrapperMaskedSecret.MaskedEntry, error) {
	return w.MaskSecrets(fileContent)
}

func (c SastChatHTTPWrapper) Call(w gptWrapper.StatefulWrapper, id uuid.UUID, messages []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return w.Call(id, messages)
}

func NewSastChatWrapper() SastChatWrapper {
	return SastChatHTTPWrapper{}
}
