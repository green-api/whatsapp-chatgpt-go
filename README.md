# WhatsApp GPT Bot Go Library

A modern, state-based WhatsApp bot library with OpenAI GPT integration, built on top of GREEN-API's WhatsApp chatbot library for Golang.

## Features

- OpenAI GPT model integration for intelligent responses
- Support for multiple GPT models (GPT-3.5, GPT-4, GPT-4o, O1)
- Multimodal capabilities with image processing support
- Voice message transcription
- Comprehensive message handling for various WhatsApp message types
- Middleware architecture for customizing message and response processing
- Built-in conversation history management

## Installation

```bash
go get github.com/green-api/whatsapp-chatgpt-go
```

This will also install the required dependencies:
- `github.com/green-api/whatsapp-chatbot-golang`
- `github.com/sashabaranov/go-openai`

## Quick Start

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/green-api/whatsapp-chatgpt-go"
)

func main() {
    // Initialize the bot
    bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
        IDInstance:       "your-instance-id",
        APITokenInstance: "your-token",
        OpenAIApiKey:     "your-openai-api-key",
        Model:            whatsapp_chatgpt_go.ModelGPT4o,
        SystemMessage:    "You are a helpful assistant.",
    })

    // Set up signal handling for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start the bot in a goroutine
    go func() {
        log.Println("Starting WhatsApp GPT bot...")
        bot.StartReceivingNotifications()
    }()

    // Wait for termination signal
    <-sigChan

    // Shutdown
    log.Println("Shutting down bot...")
    bot.StopReceivingNotifications()
    log.Println("Bot stopped.")
}
```

## Usage Patterns

This library supports two distinct usage patterns depending on your needs:

### 1. Standalone Bot

You can run the bot as a standalone service that listens for and processes WhatsApp messages automatically:

```go
bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    SystemMessage:    "You are a helpful assistant.",
})

// Start listening for webhooks and processing messages
bot.StartReceivingNotifications()
```

### 2. Message Processor

Alternatively, you can use the bot as a message processing utility within your own application:

```go
gptBot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    SystemMessage:    "You are a helpful assistant.",
})

// No need to call StartReceivingNotifications - just use ProcessMessage when needed
response, updatedSessionData, err := gptBot.ProcessMessage(
    ctx, 
    notification,  // The notification from your own webhook handling
    sessionData    // Your own session data
)
if err != nil {
    // Handle error
}

// Handle the response in your own way
// Store the updated session data in your own state system
```

## Core Components

### Bot Configuration

Complete configuration options for the WhatsappGptBot:

```go
type GPTBotConfig struct {
    // OpenAI API key
    OpenAIApiKey string
    
    // Model to use (default: gpt-4o)
    Model OpenAIModel
    
    // Maximum number of messages to keep in history (default: 10)
    MaxHistoryLength int
    
    // System message to set the bot's personality
    SystemMessage string
    
    // Temperature for response generation (default: 0.5)
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
```

### WhatsappGptBot

Main struct for creating and managing your OpenAI-powered WhatsApp bot:

```go
bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    // Required parameters
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",

    // Optional GPT-specific parameters
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    MaxHistoryLength: 15,
    SystemMessage:    "You are a helpful assistant specializing in customer support.",
    Temperature:      0.7,
    ErrorMessage:     "Sorry, I couldn't process your request. Please try again.",
    
    // Optional behavior parameters
    ClearWebhookQueueOnStart: true,
})
```

## Message Handling

The bot automatically handles different types of WhatsApp messages and converts them into a format understood by OpenAI's models.

### Supported Message Types

- **Text**
- **Image**
- **Audio**
- **Video**
- **Document**
- **Poll**
- **Location**
- **Contact**

### Message Handler Registry

The bot uses a registry of message handlers to process different message types:

```go
// Create a custom message handler
type CustomMessageHandler struct{}

func (h *CustomMessageHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
    // Logic to determine if this handler can process the message
    return true
}

func (h *CustomMessageHandler) ProcessMessage(
    notification *whatsapp_chatbot_golang.Notification,
    client *openai.Client,
    model whatsapp_chatgpt_go.OpenAIModel,
) (interface{}, error) {
    // Process the message
    return "Processed content", nil
}

// Register the custom handler
bot.RegisterMessageHandler(&CustomMessageHandler{})
```

## Middleware System

The middleware system allows for customizing message processing before sending to GPT and response processing before sending back to the user.

### Adding Message Middleware

```go
// Process messages before sending to GPT
bot.AddMessageMiddleware(func(
    notification *whatsapp_chatbot_golang.Notification,
    messageContent interface{},
    messages []openai.ChatCompletionMessage,
    sessionData *whatsapp_chatgpt_go.GPTSessionData,
) (interface{}, []openai.ChatCompletionMessage, error) {
    // Add custom context or modify the message
    sender, _ := notification.Sender()
    log.Printf("Processing message from %s: %v", sender, messageContent)
    
    return messageContent, messages, nil
})
```

### Adding Response Middleware

```go
// Process GPT responses before sending to user
bot.AddResponseMiddleware(func(
    response string,
    messages []openai.ChatCompletionMessage,
    sessionData *whatsapp_chatgpt_go.GPTSessionData,
) (string, []openai.ChatCompletionMessage, error) {
    // Format or modify the response
    formattedResponse := response + "\n\n_Powered by GREEN-API_"
    
    return formattedResponse, messages, nil
})
```

## Session Data

The GPT bot extends the base session data with conversation-specific information:

```go
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
```

You can access and modify this data in your middleware or through the available methods.

## Supported OpenAI Models

The library supports a variety of OpenAI models:

### GPT-4 Models

- ModelGPT4 ("gpt-4")
- ModelGPT4Turbo ("gpt-4-turbo")
- ModelGPT4TurboPreview ("gpt-4-turbo-preview")
- ModelGPT41106Preview ("gpt-4-1106-preview")
- ModelGPT40125Preview ("gpt-4-0125-preview")
- ModelGPT432k ("gpt-4-32k")

### GPT-4o Models

- ModelGPT4o ("gpt-4o") - default
- ModelGPT4oMini ("gpt-4o-mini")

### GPT-3.5 Models

- ModelGPT35Turbo ("gpt-3.5-turbo")
- ModelGPT35Turbo16k ("gpt-3.5-turbo-16k")
- ModelGPT35Turbo1106 ("gpt-3.5-turbo-1106")
- ModelGPT35Turbo0125 ("gpt-3.5-turbo-0125")

### o1 Models

- ModelO1 ("o1")
- ModelO1Mini ("o1-mini")
- ModelO1Preview ("o1-preview")

### Image-Capable Models

The following models can process images:

- ModelGPT4o ("gpt-4o")
- ModelGPT4oMini ("gpt-4o-mini")
- ModelGPT4Turbo ("gpt-4-turbo")

You can check if a model supports images using:

```go
if whatsapp_chatgpt_go.SupportsImages(bot.GetModel()) {
    // Handle image-based workflow
}
```

## Complete Bot Example with Middleware

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-chatgpt-go"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
)

func truncateString(s string, length int) string {
	if len(s) > length {
		return s[:length] + "..."
	}
	return s
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	idInstance := os.Getenv("GREEN_API_ID_INSTANCE")
	apiTokenInstance := os.Getenv("GREEN_API_TOKEN_INSTANCE")
	openaiToken := os.Getenv("OPENAI_API_KEY")

	if idInstance == "" || apiTokenInstance == "" || openaiToken == "" {
		log.Fatalf("Missing required environment variables: GREEN_API_ID_INSTANCE, GREEN_API_TOKEN_INSTANCE, OPENAI_API_KEY")
	}

	config := whatsapp_chatgpt_go.GPTBotConfig{
		IDInstance:               idInstance,
		APITokenInstance:         apiTokenInstance,
		OpenAIApiKey:             openaiToken,
		Model:                    whatsapp_chatgpt_go.ModelGPT4o,
		MaxHistoryLength:         10,
		SystemMessage:            "You are a helpful assistant responding via WhatsApp.",
		Temperature:              0.7,
		ErrorMessage:             "Sorry, I encountered an error processing your message.",
		ClearWebhookQueueOnStart: true,
	}

	bot := whatsapp_chatgpt_go.NewWhatsappGptBot(config)

	// Example Middleware: Logs details about incoming message processing.
	bot.AddMessageMiddleware(func(notification *whatsapp_chatbot_golang.Notification,
		messageContent interface{},
		messages []openai.ChatCompletionMessage,
		sessionData *whatsapp_chatgpt_go.GPTSessionData) (interface{}, []openai.ChatCompletionMessage, error) {

		sender, _ := notification.Sender()

		var contentLog string
		if parts, ok := messageContent.([]openai.ChatMessagePart); ok {
			contentLog = "MultiContent Parts: ["
			for i, p := range parts {
				if i > 0 {
					contentLog += ", "
				}
				contentLog += fmt.Sprintf("{Type: %s, ", p.Type)
				if p.Type == openai.ChatMessagePartTypeText {
					contentLog += fmt.Sprintf("Text: '%s'", p.Text)
				} else if p.Type == openai.ChatMessagePartTypeImageURL && p.ImageURL != nil {
					urlStr := p.ImageURL.URL
					if len(urlStr) > 50 {
						urlStr = urlStr[:47] + "..."
					}
					contentLog += fmt.Sprintf("ImageURL: %s", urlStr)
				} else {
					contentLog += "OtherPartData"
				}
				contentLog += "}"
			}
			contentLog += "]"
		} else {
			contentLog = fmt.Sprintf("Text Content: '%s'", truncateString(fmt.Sprintf("%v", messageContent), 100))
		}
		log.Printf("--> MID: Received from %s: %s", sender, contentLog)
		log.Printf("--> MID: History has %d messages before adding current.", len(messages))

		return messageContent, messages, nil
	})

	// Example Middleware: Logs the response being sent.
	bot.AddResponseMiddleware(func(response string,
		messages []openai.ChatCompletionMessage,
		sessionData *whatsapp_chatgpt_go.GPTSessionData) (string, []openai.ChatCompletionMessage, error) {

		log.Printf("<-- MID: Sending response: %s", truncateString(response, 100))
		log.Printf("<-- MID: History has %d messages after adding assistant response.", len(messages))
		return response, messages, nil
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Starting WhatsApp GPT bot...")
		bot.StartReceivingNotifications()
		log.Println("Notification receiving loop stopped.")
	}()

	<-sigChan

	log.Println("Shutting down bot...")
	bot.StopReceivingNotifications()
	log.Println("Bot stopped.")
}
```

## Available Methods

### WhatsappGptBot

- `NewWhatsappGptBot(config GPTBotConfig) *WhatsappGptBot` - Creates a new GPT-enabled WhatsApp bot
- `StartReceivingNotifications()` - Starts receiving and processing webhook notifications
- `StopReceivingNotifications()` - Stops the notification listener
- `ProcessMessage(ctx context.Context, notification *whatsapp_chatbot_golang.Notification, sessionData *GPTSessionData) (string, *GPTSessionData, error)` - Processes a message without using the bot's internal state manager
- `AddMessageMiddleware(middleware ProcessMessageMiddleware)` - Registers a middleware to process incoming messages
- `AddResponseMiddleware(middleware ProcessResponseMiddleware)` - Registers a middleware to process GPT responses
- `RegisterMessageHandler(handler MessageHandler)` - Adds a custom message handler
- `GetOpenAI() *openai.Client` - Returns the OpenAI client instance
- `GetModel() OpenAIModel` - Returns the configured Open AI model identifier
- `GetSystemMessage() string` - Returns the configured system message
- `SupportsImages() bool` - Checks if the currently configured model supports image input
- `Methods()` - Access to the base library's methods for sending messages, etc.

## License

MIT