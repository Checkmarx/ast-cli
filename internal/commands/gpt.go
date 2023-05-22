package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func NewGPTCommand(kicsGptWrapper wrappers.KicsGptWrapper) *cobra.Command {
	kicsGptcommand := &cobra.Command{
		Use:   "kics-gpt",
		Short: "kics-gpt",
		Long:  "kics-gpt.",
	}
	chatCmd := chatSubCommand(kicsGptWrapper)

	kicsGptcommand.AddCommand(chatCmd)
	return kicsGptcommand
}

func chatSubCommand(kicsGptWrapper wrappers.KicsGptWrapper) *cobra.Command {
	kicsChatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat.",

		RunE: RunKicsChat(kicsGptWrapper),
	}

	kicsChatCmd.PersistentFlags().String(params.GptToken, "", "token")
	kicsChatCmd.PersistentFlags().String(params.ConversationIdFlag, "", "conversationId")
	kicsChatCmd.PersistentFlags().String(params.MessageFlag, "", "Message")

	return kicsChatCmd
}

func RunKicsChat(kicsGptWrapper wrappers.KicsGptWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		conversationID, _ := cmd.Flags().GetString(params.ConversationIdFlag)
		userMessage, _ := cmd.Flags().GetString(params.MessageFlag)
		token, _ := cmd.Flags().GetString(params.GptToken)

		if conversationID == "" {
			// Generate conversation ID if not provided
			conversationID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		// Load conversation from file if it exists
		conv, err := loadConversation(conversationID)
		if err != nil {
			return err
		}

		// Add user message to conversation
		conv.Messages = append(conv.Messages, wrappers.Message{
			User: "user",
			Text: userMessage,
			Time: time.Now(),
		})

		// Send conversation to ChatGPT and get response
		response, err := kicsGptWrapper.SendToChatGPT(conv, token)
		if err != nil {
			return err
		}

		conv.Messages = append(conv.Messages, wrappers.Message{
			User: "assistant",
			Text: response,
			Time: time.Now(),
		})

		// Save conversation to file
		err = saveConversation(conv)
		if err != nil {
			return err
		}

		// Print conversation as JSON
		jsonBytes, err := json.Marshal(conv)
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))

		return nil
	}
}

func loadConversation(conversationID string) (*wrappers.Conversation, error) {
	filePath := getConversationFilePath(conversationID)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Conversation file does not exist, return new conversation
		return &wrappers.Conversation{
			ID:       conversationID,
			Messages: []wrappers.Message{},
		}, nil
	}

	// Load conversation from file
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var conv wrappers.Conversation
	err = json.Unmarshal(bytes, &conv)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func saveConversation(conv *wrappers.Conversation) error {
	jsonBytes, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return err
	}
	filePath := getConversationFilePath(conv.ID)

	return ioutil.WriteFile(filePath, jsonBytes, 0644)
}

func getConversationFilePath(conversationID string) string {
	path := filepath.Join(".", "conversations")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0644)
	}
	return filepath.Join(path, conversationID+".json")
}
