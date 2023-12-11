package wrappers

import (
	//gptWrapperMaskedSecret "github.com/checkmarxDev/gpt-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/checkmarxDev/gpt-wrapper/pkg/message"
	gptWrapper "github.com/checkmarxDev/gpt-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type SastChatWrapper interface {
	Call(gptWrapper.StatefulWrapper, uuid.UUID, []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error)
}
