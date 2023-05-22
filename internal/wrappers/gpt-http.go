package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

type KicsHTTPGptWrapper struct {
	client *http.Client
}

func NewKicsGptWrapper() KicsGptWrapper {
	return &KicsHTTPGptWrapper{
		client: GetClient(viper.GetUint(params.ClientTimeoutKey)),
	}
}

func (g *KicsHTTPGptWrapper) SendToChatGPT(conv *Conversation, token string) (string, error) {
	// Convert conversation to ChatGPT input format
	var requestBody RequestBody
	requestBody.Model = "gpt-3.5-turbo"

	for _, msg := range conv.Messages {
		requestBody.Messages = append(requestBody.Messages, RequestMessage{
			Role:    msg.User,
			Content: msg.Text,
		})
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Response: %s", jsonData))

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Response: %s", bodyBytes))

	// Parse ChatGPT response
	var chatGPTResp ChatGPTResponse
	err = json.Unmarshal(bodyBytes, &chatGPTResp)
	if err != nil {
		return "", err
	}

	return chatGPTResp.Choices[0].Message.Content, nil
}
