package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	gptWrapperMaskedSecret "github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	gptWrapperMessage "github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	gptWrapper "github.com/Checkmarx/gen-ai-wrapper/pkg/wrapper"
	"github.com/google/uuid"
	"k8s.io/kubectl/pkg/util/slice"
)

const (
	openAIurl = "https://api.openai.com/v1/models"
)

type ChatHTTPWrapper struct {
}

func NewChatWrapper() ChatWrapper {
	return ChatHTTPWrapper{}
}

func (c ChatHTTPWrapper) MaskSecrets(w gptWrapper.StatefulWrapper, fileContent string) (*gptWrapperMaskedSecret.MaskedEntry, error) {
	return w.MaskSecrets(fileContent)
}

func (c ChatHTTPWrapper) Call(w gptWrapper.StatefulWrapper, id uuid.UUID, messages []gptWrapperMessage.Message) ([]gptWrapperMessage.Message, error) {
	return w.Call(id, messages)
}

func (c ChatHTTPWrapper) SecureCall(w gptWrapper.StatefulWrapper, historyID uuid.UUID, messages []gptWrapperMessage.Message, metaData *gptWrapperMessage.MetaData, cxAuth string) (
	[]gptWrapperMessage.Message,
	error,
) {
	return w.SecureCall(cxAuth, metaData, historyID, messages)
}

func (c ChatHTTPWrapper) GetModelList(apiKey string, ownedBy []string) ViewOpenAiModels {

	var viewOpenAiModels ViewOpenAiModels
	response, err := SendHTTPRequestByFullURL(http.MethodGet, openAIurl, nil, true, 10, apiKey, false)
	if err != nil {
		fmt.Println("error here")
		viewOpenAiModels.Error = err.Error()
		return viewOpenAiModels
	}

	switch response.StatusCode {
	case http.StatusUnauthorized:
		viewOpenAiModels.Error = "unauthorized, incorrect API key provided"
	case http.StatusNotFound:
		viewOpenAiModels.Error = "url not found to fetch list of models"
	}

	body, _ := io.ReadAll(response.Body)
	var openAIresponse OpenAIresponse
	json.Unmarshal(body, &openAIresponse)

	if len(openAIresponse.Data) > 0 {
		sort.Slice(openAIresponse.Data, func(i, j int) bool {
			return openAIresponse.Data[i].Created > openAIresponse.Data[j].Created
		})
		for _, model := range openAIresponse.Data {
			if len(ownedBy) == 0 {
				viewOpenAiModels.Models = append(viewOpenAiModels.Models, model.Id)
			} else {
				slice.ContainsString(ownedBy, model.Owned_by, nil)
			}

		}
	}

	return viewOpenAiModels
}
