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
