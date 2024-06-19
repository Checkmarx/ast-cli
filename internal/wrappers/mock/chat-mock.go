package mock

import (
	gptWrapperMaskedSecret "github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	gptWrapperRole "github.com/Checkmarx/gen-ai-wrapper/pkg/role"
	gptWrapper "github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatMockWrapper struct {
}

func (c ChatMockWrapper) MaskSecrets(wrapper gptWrapper.StatefulWrapper, s string) (*gptWrapperMaskedSecret.MaskedEntry, error) {
	return &gptWrapperMaskedSecret.MaskedEntry{
		MaskedFile: "",
		MaskedSecrets: []gptWrapperMaskedSecret.MaskedSecret{{
			Masked: "password=<masked>",
			Secret: "password=<not_masked>",
			Line:   0,
		}},
	}, nil
}

func (c ChatMockWrapper) Call(_ gptWrapper.StatefulWrapper, _ uuid.UUID, _ []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return []gptWrapperMessage.Message{{
		Role:    gptWrapperRole.Assistant,
		Content: "Mock message",
	}}, nil
}

func (c ChatMockWrapper) SecureCall(w gptWrapper.StatefulWrapper, id uuid.UUID, messages []gptWrapperMessage.Message, metaData *gptWrapperMessage.MetaData, cxAuth string) (
	[]gptWrapperMessage.Message,
	error,
) {
	return []gptWrapperMessage.Message{{
		Role:    gptWrapperRole.Assistant,
		Content: "Mock message from SecureCall",
	}}, nil
}
