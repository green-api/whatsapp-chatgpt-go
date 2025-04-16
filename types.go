package whatsapp_chatgpt_go

import (
	"github.com/green-api/whatsapp-chatbot-golang"
	"github.com/sashabaranov/go-openai"
)

// GPTSessionData contains conversation history and model settings
type GPTSessionData struct {
	// Messages in the conversation
	Messages []openai.ChatCompletionMessage `json:"messages"`
	// Timestamp of last activity
	LastActivity int64 `json:"lastActivity"`
	// Custom user data
	UserData map[string]interface{} `json:"userData,omitempty"`
	// Context for the current conversation
	Context map[string]interface{} `json:"context,omitempty"`
}

// MessageHandler interface for processing different message types
type MessageHandler interface {
	CanHandle(notification *whatsapp_chatbot_golang.Notification) bool
	ProcessMessage(notification *whatsapp_chatbot_golang.Notification, client *openai.Client, model OpenAIModel) (interface{}, error)
}

// ProcessMessageMiddleware processes a message before sending to GPT
type ProcessMessageMiddleware func(notification *whatsapp_chatbot_golang.Notification, messageContent interface{}, messages []openai.ChatCompletionMessage, sessionData *GPTSessionData) (interface{}, []openai.ChatCompletionMessage, error)

// ProcessResponseMiddleware processes a response before sending to the user
type ProcessResponseMiddleware func(response string, messages []openai.ChatCompletionMessage, sessionData *GPTSessionData) (string, []openai.ChatCompletionMessage, error)

// GPTBotConfig contains configuration for the GPT bot
type GPTBotConfig struct {
	// OpenAI API key
	OpenAIApiKey string
	// Model to use (default: gpt-4o)
	Model OpenAIModel
	// Maximum number of messages to keep in history
	MaxHistoryLength int
	// System message to set the bot's personality
	SystemMessage string
	// Temperature for response generation
	Temperature float32
	// Error message to show when something goes wrong
	ErrorMessage string
	// ID Instance from GREEN-API
	IDInstance string
	// API Token Instance from GREEN-API
	APITokenInstance string
	// Whether to clear webhook queue on start
	ClearWebhookQueueOnStart bool
}
