package wrappers

import (
	gptWrapperMessage "github.com/checkmarxDev/gpt-wrapper/pkg/message"
	gptWrapper "github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatHTTPWrapper struct {
}

func (c ChatHTTPWrapper) Call(w gptWrapper.StatefulWrapper, id uuid.UUID, messages []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return w.Call(id, messages)
}

func NewChatWrapper() ChatWrapper {
	return ChatHTTPWrapper{}
}
