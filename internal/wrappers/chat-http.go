package wrappers

import (
	gptWrapperMaskedSecret "github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	gptWrapper "github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
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

func (c ChatHTTPWrapper) SecureCall(w gptWrapper.StatefulWrapper, historyId uuid.UUID, messages []gptWrapperMessage.Message, metaData *gptWrapperMessage.MetaData, cxAuth string) (
	[]gptWrapperMessage.Message,
	error,
) {
	return w.SecureCall(cxAuth, metaData, historyId, messages)
}

func NewChatWrapper() ChatWrapper {
	return ChatHTTPWrapper{}
}
