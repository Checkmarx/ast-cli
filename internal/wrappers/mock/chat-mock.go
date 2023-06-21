package mock

import (
	gptWrapperMessage "github.com/checkmarxDev/gpt-wrapper/pkg/message"
	gptWrapperRole "github.com/checkmarxDev/gpt-wrapper/pkg/role"
	gptWrapper "github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatMockWrapper struct {
}

func (c ChatMockWrapper) Call(_ gptWrapper.StatefulWrapper, _ uuid.UUID, _ []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return []gptWrapperMessage.Message{{
		Role:    gptWrapperRole.Assistant,
		Content: "Mock message",
	}}, nil
}
