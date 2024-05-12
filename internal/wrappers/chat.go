package wrappers

import (
	gptWrapperMaskedSecret "github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	gptWrapper "github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatWrapper interface {
	Call(gptWrapper.StatefulWrapper, uuid.UUID, []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error)
	MaskSecrets(gptWrapper.StatefulWrapper, string) (*gptWrapperMaskedSecret.MaskedEntry, error)
}
