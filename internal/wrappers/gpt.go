package wrappers

import "time"

type Message struct {
	User string    `json:"user"`
	Text string    `json:"text"`
	Time time.Time `json:"time"`
}

type RequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Messages []RequestMessage `json:"messages"`
	Model    string           `json:"model"`
}

type Conversation struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

type ChatGPTResponse struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int    `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Message struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"message"`
		Text         string      `json:"text,omitempty"`
		Index        int         `json:"index,omitempty"`
		Logprobs     interface{} `json:"logprobs,omitempty"`
		FinishReason string      `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

type KicsGptWrapper interface {
	SendToChatGPT(conversation *Conversation, token string) (string, error)
}
