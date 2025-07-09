package wrappers

import (
	gptWrapperMaskedSecret "github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	gptWrapper "github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
	"github.com/google/uuid"
)

type ChatWrapper interface {
	Call(gptWrapper.StatefulWrapper, uuid.UUID, []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error)
	SecureCall(gptWrapper.StatefulWrapper, uuid.UUID, []gptWrapperMessage.Message, *gptWrapperMessage.MetaData, string) ([]gptWrapperMessage.Message, error)
	MaskSecrets(gptWrapper.StatefulWrapper, string) (*gptWrapperMaskedSecret.MaskedEntry, error)
	GetModelList(string, []string) ViewOpenAiModels
}

type OpenAIresponse struct {
	Error  interface{}  `json:"error"`
	Data   []OpenModels `json:"data"`
	Object string
}

type OpenModels struct {
	Id       string `json:"id"`
	Object   string `json:"object"`
	Created  int    `json:"created"`
	Owned_by string `json:"owned_by"`
}

type ViewOpenAiModels struct {
	Models []string `json:"models,omitempty"`
	Error  string   `json:"error,omitempty"`
}
