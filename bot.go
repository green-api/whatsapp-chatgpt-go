package whatsapp_chatgpt_go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/green-api/whatsapp-chatbot-golang"
	"github.com/sashabaranov/go-openai"
)

// WhatsappGptBot extends the base WhatsApp bot with GPT capabilities
type WhatsappGptBot struct {
	*whatsapp_chatbot_golang.Bot
	openai              *openai.Client
	maxHistoryLength    int
	systemMessage       string
	model               OpenAIModel
	temperature         float32
	errorMessage        string
	messageHandlers     []MessageHandler
	messageMiddlewares  []ProcessMessageMiddleware
	responseMiddlewares []ProcessResponseMiddleware
}

// NewWhatsappGptBot creates a new GPT-enabled WhatsApp bot
func NewWhatsappGptBot(config GPTBotConfig) *WhatsappGptBot {
	if config.Model == "" {
		config.Model = ModelGPT4o
	}
	if config.MaxHistoryLength <= 0 {
		config.MaxHistoryLength = 10
	}
	if config.Temperature <= 0 {
		config.Temperature = 0.5
	}
	if config.ErrorMessage == "" {
		config.ErrorMessage = "Sorry, I encountered an error processing your message. Please try again."
	}

	baseBot := whatsapp_chatbot_golang.NewBot(config.IDInstance, config.APITokenInstance)
	client := openai.NewClient(config.OpenAIApiKey)

	gptBot := &WhatsappGptBot{
		Bot:                 baseBot,
		openai:              client,
		maxHistoryLength:    config.MaxHistoryLength,
		systemMessage:       config.SystemMessage,
		model:               config.Model,
		temperature:         config.Temperature,
		errorMessage:        config.ErrorMessage,
		messageHandlers:     []MessageHandler{},
		messageMiddlewares:  []ProcessMessageMiddleware{},
		responseMiddlewares: []ProcessResponseMiddleware{},
	}

	gptBot.initDefaultHandlers()
	gptBot.IncomingMessageHandler(gptBot.handleIncomingMessage)
	gptBot.CleanNotificationQueue = config.ClearWebhookQueueOnStart

	return gptBot
}

func (bot *WhatsappGptBot) initDefaultHandlers() {
	bot.RegisterMessageHandler(&TextHandler{})
	bot.RegisterMessageHandler(&ImageHandler{})
	bot.RegisterMessageHandler(&AudioHandler{})
	bot.RegisterMessageHandler(&VideoHandler{})
	bot.RegisterMessageHandler(&DocumentHandler{})
	bot.RegisterMessageHandler(&LocationHandler{})
	bot.RegisterMessageHandler(&ContactHandler{})
	bot.RegisterMessageHandler(&PollHandler{})
	bot.RegisterMessageHandler(&PollUpdateHandler{})
	bot.RegisterMessageHandler(&FallbackHandler{})
}

func (bot *WhatsappGptBot) handleIncomingMessage(notification *whatsapp_chatbot_golang.Notification) {
	typeWebhook, ok := notification.Body["typeWebhook"].(string)
	if !ok || typeWebhook != "incomingMessageReceived" {
		return
	}

	sessionData := bot.getOrCreateSession(notification)
	ctx := context.Background()

	response, err := bot.processMessageWithState(ctx, notification, sessionData)
	if err != nil {
		log.Printf("Error processing message for chat %s: %v", notification.StateId, err)
		_, sendErr := bot.Sending().SendMessage(notification.StateId, bot.errorMessage)
		if sendErr != nil {
			log.Printf("Error sending error message to %s: %v", notification.StateId, sendErr)
		}
		return
	}

	_, sendErr := bot.Sending().SendMessage(notification.StateId, response)
	if sendErr != nil {
		log.Printf("Error sending GPT response to %s: %v", notification.StateId, sendErr)
	}
}

// getOrCreateSession retrieves or initializes session data from the state manager.
func (bot *WhatsappGptBot) getOrCreateSession(notification *whatsapp_chatbot_golang.Notification) *GPTSessionData {
	stateData := notification.GetStateData()
	var sessionData GPTSessionData

	if stateData != nil {
		if jsonString, ok := stateData["gptSessionJson"].(string); ok && jsonString != "" {
			err := json.Unmarshal([]byte(jsonString), &sessionData)
			if err == nil {
				if len(sessionData.Messages) == 0 || sessionData.Messages[0].Role != openai.ChatMessageRoleSystem || sessionData.Messages[0].Content != bot.systemMessage {
					log.Printf("Warning: Loaded session for %s has invalid/missing system message. Re-prepending.", notification.StateId)
					validMessages := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: bot.systemMessage}}
					for _, msg := range sessionData.Messages {
						if msg.Role != openai.ChatMessageRoleSystem {
							validMessages = append(validMessages, msg)
						}
					}
					sessionData.Messages = validMessages
				}
				return &sessionData
			} else {
				log.Printf("Error unmarshalling gptSessionJson for %s: %v. Creating new session.", notification.StateId, err)
			}
		} else {
			if oldSessionMap, ok := stateData["gptSession"].(map[string]interface{}); ok {
				log.Printf("Warning: Found old map-based session for %s. Migrating to JSON format (multi-modal history might be lost for this specific load).", notification.StateId)
				migratedSession := mapToGptSessionDataInternal(oldSessionMap, bot.systemMessage)
				bot.updateSessionState(notification, migratedSession)
				return migratedSession
			}
		}
	}

	log.Printf("Creating new GPT session (JSON format) for %s", notification.StateId)
	sessionData = GPTSessionData{
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: bot.systemMessage},
		},
		LastActivity: time.Now().Unix(),
		UserData:     make(map[string]interface{}),
		Context:      make(map[string]interface{}),
	}

	bot.updateSessionState(notification, &sessionData)

	return &sessionData
}

// --- mapToGptSessionDataInternal (Helper ONLY for migration from old format) ---
func mapToGptSessionDataInternal(sessionMap map[string]interface{}, systemMessage string) *GPTSessionData {
	sessionData := &GPTSessionData{
		Messages:     []openai.ChatCompletionMessage{},
		LastActivity: 0,
		UserData:     make(map[string]interface{}),
		Context:      make(map[string]interface{}),
	}

	if msgMapsData, ok := sessionMap["messages"]; ok {
		if msgMaps, ok := msgMapsData.([]interface{}); ok {
			for _, msgMapInterface := range msgMaps {
				if msgMap, ok := msgMapInterface.(map[string]interface{}); ok {
					role, _ := msgMap["role"].(string)
					content, _ := msgMap["content"].(string)
					if role != "" {
						sessionData.Messages = append(sessionData.Messages, openai.ChatCompletionMessage{
							Role:    role,
							Content: content,
						})
					}
				}
			}
		}
	}

	if len(sessionData.Messages) == 0 || sessionData.Messages[0].Role != openai.ChatMessageRoleSystem {
		sessionData.Messages = []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemMessage},
		}
	}

	if lastActivity, ok := sessionMap["lastActivity"].(float64); ok {
		sessionData.LastActivity = int64(lastActivity)
	} else if lastActivityInt, ok := sessionMap["lastActivity"].(int64); ok {
		sessionData.LastActivity = lastActivityInt
	}

	if userData, ok := sessionMap["userData"].(map[string]interface{}); ok {
		sessionData.UserData = userData
	}
	if contextData, ok := sessionMap["context"].(map[string]interface{}); ok {
		sessionData.Context = contextData
	}

	return sessionData
}

// processMessageWithState handles message processing using the bot's internal state.
func (bot *WhatsappGptBot) processMessageWithState(ctx context.Context, notification *whatsapp_chatbot_golang.Notification, sessionData *GPTSessionData) (string, error) {
	messageContent, err := bot.processMessageContent(notification)
	if err != nil {
		return "", fmt.Errorf("failed to process message content: %w", err)
	}

	response, updatedMessages, err := bot.generateGPTResponse(ctx, notification, messageContent, sessionData.Messages, sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to generate GPT response: %w", err)
	}

	sessionData.Messages = updatedMessages
	bot.updateSessionState(notification, sessionData)

	return response, nil
}

// ProcessMessage allows processing a message without using the bot's internal state manager.
func (bot *WhatsappGptBot) ProcessMessage(ctx context.Context, notification *whatsapp_chatbot_golang.Notification, sessionData *GPTSessionData) (string, *GPTSessionData, error) {
	if sessionData == nil {
		log.Println("Warning: ProcessMessage called with nil sessionData, initializing.")
		sessionData = &GPTSessionData{
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: bot.systemMessage},
			},
			LastActivity: time.Now().Unix(),
			UserData:     make(map[string]interface{}),
			Context:      make(map[string]interface{}),
		}
	}

	messageContent, err := bot.processMessageContent(notification)
	if err != nil {
		return "", sessionData, fmt.Errorf("failed to process message content: %w", err)
	}

	response, updatedMessages, err := bot.generateGPTResponse(ctx, notification, messageContent, sessionData.Messages, sessionData)
	if err != nil {
		return "", sessionData, fmt.Errorf("failed to generate GPT response: %w", err)
	}

	updatedSessionData := &GPTSessionData{
		Messages:     updatedMessages,
		LastActivity: time.Now().Unix(),
		UserData:     sessionData.UserData,
		Context:      sessionData.Context,
	}

	return response, updatedSessionData, nil
}

// generateGPTResponse is the core logic for interacting with OpenAI API.
func (bot *WhatsappGptBot) generateGPTResponse(
	ctx context.Context,
	notification *whatsapp_chatbot_golang.Notification,
	initialMessageContent interface{},
	initialMessages []openai.ChatCompletionMessage,
	sessionData *GPTSessionData,
) (string, []openai.ChatCompletionMessage, error) {

	messageContent := initialMessageContent
	messages := make([]openai.ChatCompletionMessage, len(initialMessages))
	copy(messages, initialMessages)

	for _, middleware := range bot.messageMiddlewares {
		var middlewareErr error
		messageContent, messages, middlewareErr = middleware(notification, messageContent, messages, sessionData)
		if middlewareErr != nil {
			return "", initialMessages, fmt.Errorf("message middleware failed: %w", middlewareErr)
		}
	}

	var userMsg openai.ChatCompletionMessage
	if parts, ok := messageContent.([]openai.ChatMessagePart); ok {
		userMsg = openai.ChatCompletionMessage{
			Role:         openai.ChatMessageRoleUser,
			MultiContent: parts,
		}
	} else {
		userMsg = openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: toString(messageContent),
		}
	}
	messages = append(messages, userMsg)

	if len(messages) > bot.maxHistoryLength+1 {
		systemMsg := messages[0]
		startIndex := len(messages) - bot.maxHistoryLength
		messages = append([]openai.ChatCompletionMessage{systemMsg}, messages[startIndex:]...)
	}

	resp, err := bot.openai.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       bot.model.String(),
			Messages:    messages,
			Temperature: bot.temperature,
		},
	)
	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) {
			paramStr := "<nil>"
			if apiErr.Param != nil {
				paramStr = *apiErr.Param
			}
			log.Printf("OpenAI API Error Details - Type: %s, Code: %v, Param: %s, Message: %s",
				apiErr.Type, apiErr.Code, paramStr, apiErr.Message)
		}
		return "", initialMessages, fmt.Errorf("openai chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		log.Println("Warning: OpenAI response missing choices or content")
		return "", initialMessages, errors.New("no response generated by OpenAI")
	}

	assistantMessage := resp.Choices[0].Message
	responseContent := assistantMessage.Content
	messages = append(messages, assistantMessage)

	finalResponseContent := responseContent
	for _, middleware := range bot.responseMiddlewares {
		var middlewareErr error
		finalResponseContent, messages, middlewareErr = middleware(finalResponseContent, messages, sessionData)
		if middlewareErr != nil {
			return "", messages[:len(messages)-1], fmt.Errorf("response middleware failed: %w", middlewareErr)
		}
	}

	if finalResponseContent != responseContent {
		messages[len(messages)-1].Content = finalResponseContent
	}

	return finalResponseContent, messages, nil
}

// processMessageContent iterates through handlers to find one that can process the notification.
func (bot *WhatsappGptBot) processMessageContent(notification *whatsapp_chatbot_golang.Notification) (interface{}, error) {
	for _, handler := range bot.messageHandlers {
		if handler.CanHandle(notification) {
			content, err := handler.ProcessMessage(notification, bot.openai, bot.model)
			if err != nil {
				log.Printf("Error processing message with handler %T: %v", handler, err)
				return fmt.Sprintf("[Error processing message type %T: %v]", handler, err), nil
			}
			return content, nil
		}
	}
	log.Printf("Error: No suitable handler found for message type (FallbackHandler missing or failed?). Body: %v", notification.Body)
	return "[Internal Error: Could not find a handler for this message]", errors.New("no suitable message handler found")
}

// updateSessionState saves the updated session data back to the state manager.
func (bot *WhatsappGptBot) updateSessionState(notification *whatsapp_chatbot_golang.Notification, sessionData *GPTSessionData) {
	sessionData.LastActivity = time.Now().Unix()

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		log.Printf("CRITICAL: Failed to marshal session data for %s: %v. Session state may be lost.", notification.StateId, err)
		return
	}

	stateData := notification.GetStateData()
	if stateData == nil {
		stateData = make(map[string]interface{})
	}

	stateData["gptSessionJson"] = string(jsonData)
	delete(stateData, "gptSession")

	notification.SetStateData(stateData)
}

// AddMessageMiddleware registers a middleware to process incoming messages before GPT.
func (bot *WhatsappGptBot) AddMessageMiddleware(middleware ProcessMessageMiddleware) {
	bot.messageMiddlewares = append(bot.messageMiddlewares, middleware)
}

// AddResponseMiddleware registers a middleware to process GPT responses before sending.
func (bot *WhatsappGptBot) AddResponseMiddleware(middleware ProcessResponseMiddleware) {
	bot.responseMiddlewares = append(bot.responseMiddlewares, middleware)
}

// RegisterMessageHandler adds a custom message handler.
func (bot *WhatsappGptBot) RegisterMessageHandler(handler MessageHandler) {
	fallbackIndex := -1
	for i, h := range bot.messageHandlers {
		if _, ok := h.(*FallbackHandler); ok {
			fallbackIndex = i
			break
		}
	}
	if fallbackIndex != -1 {
		bot.messageHandlers = append(bot.messageHandlers[:fallbackIndex], append([]MessageHandler{handler}, bot.messageHandlers[fallbackIndex:]...)...)
	} else {
		bot.messageHandlers = append(bot.messageHandlers, handler)
	}
}

// GetOpenAI returns the OpenAI client instance for potential custom use.
func (bot *WhatsappGptBot) GetOpenAI() *openai.Client {
	return bot.openai
}

// GetModel returns the configured OpenAI model identifier.
func (bot *WhatsappGptBot) GetModel() OpenAIModel {
	return bot.model
}

// GetSystemMessage returns the configured system message for the bot.
func (bot *WhatsappGptBot) GetSystemMessage() string {
	return bot.systemMessage
}

// SupportsImages checks if the currently configured model supports image input.
func (bot *WhatsappGptBot) SupportsImages() bool {
	return SupportsImages(bot.model)
}

// toString converts simple message content types to a string representation.
func toString(messageContent interface{}) string {
	switch v := messageContent.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
