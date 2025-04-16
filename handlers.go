package whatsapp_chatgpt_go

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/green-api/whatsapp-chatbot-golang"
	"github.com/sashabaranov/go-openai"
)

// Helper function to safely extract string from map[string]interface{}
func getStringField(data map[string]interface{}, key string) string {
	if val, exists := data[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// Helper function to safely extract boolean from map[string]interface{}
func getBoolField(data map[string]interface{}, key string) bool {
	if val, exists := data[key]; exists {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// Helper function to safely extract slice of interfaces from map[string]interface{}
func getSliceField(data map[string]interface{}, key string) []interface{} {
	if val, exists := data[key]; exists {
		if sliceVal, ok := val.([]interface{}); ok {
			return sliceVal
		}
	}
	return nil
}

// TextHandler handles text messages
type TextHandler struct{}

// CanHandle checks if the message is a text message
func (h *TextHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "textMessage" || msgType == "extendedTextMessage"
}

// ProcessMessage processes a text message
func (h *TextHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	return notification.Text()
}

// ImageHandler handles image messages
type ImageHandler struct{}

// CanHandle checks if the message is an image message
func (h *ImageHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "imageMessage"
}

// ProcessMessage processes an image message
func (h *ImageHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, model OpenAIModel) (interface{}, error) {
	supportsVision := SupportsImages(model)

	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[User sent image, but message data missing]", nil
	}
	imageData, exists := messageData["fileMessageData"].(map[string]interface{})
	if !exists {
		return "[User sent image, but file data missing]", nil
	}

	caption := getStringField(imageData, "caption")
	urlFile := getStringField(imageData, "downloadUrl")

	if urlFile == "" {
		desc := "[User sent image, but URL missing]"
		if caption != "" {
			desc += fmt.Sprintf(" Caption: \"%s\"", caption)
		}
		return desc, nil
	}

	if supportsVision {
		textPartContent := caption
		if textPartContent == "" {
			textPartContent = " "
		}

		parts := []openai.ChatMessagePart{
			{
				Type: openai.ChatMessagePartTypeText,
				Text: textPartContent,
			},
			{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    urlFile,
					Detail: openai.ImageURLDetailAuto,
				},
			},
		}
		return parts, nil
	} else {
		var imageDescription string
		if caption != "" {
			imageDescription = fmt.Sprintf("[The user sent an image with caption: \"%s\"]", caption)
		} else {
			imageDescription = "[The user sent an image]"
		}
		return imageDescription, nil
	}
}

// AudioHandler handles audio/voice messages
type AudioHandler struct{}

// CanHandle checks if the message is an audio message
func (h *AudioHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "audioMessage" || msgType == "voiceMessage"
}

// ProcessMessage processes an audio message
func (h *AudioHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, client *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[The user sent an audio message but I couldn't process it]", nil
	}
	audioData, exists := messageData["fileMessageData"].(map[string]interface{})
	if !exists {
		return "[The user sent an audio message but I couldn't process it]", nil
	}
	urlFile := getStringField(audioData, "downloadUrl")
	if urlFile == "" {
		return "[The user sent an audio message but no URL was provided]", nil
	}

	audioFile, err := downloadAudio(urlFile)
	if err != nil {
		log.Printf("Error downloading audio: %v", err)
		return fmt.Sprintf("[The user sent an audio message but I couldn't download it: %v]", err), nil
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Printf("Warning: Failed to remove temporary audio file %s: %v", name, err)
		}
	}(audioFile)

	transcript, err := transcribeAudio(client, audioFile)
	if err != nil {
		log.Printf("Error transcribing audio: %v", err)
		return fmt.Sprintf("[The user sent an audio message. I couldn't transcribe it: %v]", err), nil
	}

	return fmt.Sprintf("User sent an audio. Transcription: \"%s\"", transcript), nil
}

// downloadAudio downloads the audio file from the URL
func downloadAudio(url string) (string, error) {
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("whatsapp-audio-%d.ogg", time.Now().UnixNano()))

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Warning: Failed to close HTTP response body for audio download: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Printf("Warning: Failed to close temporary audio file %s during write: %v", out.Name(), err)
		}
	}(out)

	_, err = io.Copy(out, resp.Body)
	return tempFile, err
}

// transcribeAudio transcribes the audio file using OpenAI's Whisper API
func transcribeAudio(client *openai.Client, audioFilePath string) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: audioFilePath,
	}

	resp, err := client.CreateTranscription(context.Background(), req)
	if err != nil {
		return "", err
	}

	if resp.Text == "" {
		return "[Audio transcription was empty]", nil
	}

	return resp.Text, nil
}

// VideoHandler handles video messages
type VideoHandler struct{}

// CanHandle checks if the message is a video message
func (h *VideoHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "videoMessage"
}

// ProcessMessage processes a video message
func (h *VideoHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a video but I couldn't process it]", nil
	}
	videoData, exists := messageData["fileMessageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a video but I couldn't process it]", nil
	}

	caption := getStringField(videoData, "caption")
	fileName := getStringField(videoData, "fileName")
	if fileName == "" {
		fileName = "video"
	}

	captionText := ""
	if caption != "" {
		captionText = fmt.Sprintf(" with caption: \"%s\"", caption)
	}

	return fmt.Sprintf("[The user sent a video: \"%s\"%s]", fileName, captionText), nil
}

// DocumentHandler handles document messages
type DocumentHandler struct{}

// CanHandle checks if the message is a document message
func (h *DocumentHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "documentMessage"
}

// ProcessMessage processes a document message
func (h *DocumentHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a document but I couldn't process it]", nil
	}
	docData, exists := messageData["fileMessageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a document but I couldn't process it]", nil
	}

	caption := getStringField(docData, "caption")
	fileName := getStringField(docData, "fileName")
	if fileName == "" {
		fileName = "document"
	}

	captionText := ""
	if caption != "" {
		captionText = fmt.Sprintf(" with caption: \"%s\"", caption)
	}

	return fmt.Sprintf("[The user sent a document: \"%s\"%s]", fileName, captionText), nil
}

// LocationHandler handles location messages
type LocationHandler struct{}

// CanHandle checks if the message is a location message
func (h *LocationHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "locationMessage"
}

// ProcessMessage processes a location message
func (h *LocationHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a location but I couldn't process it]", nil
	}
	locationData, exists := messageData["locationMessageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a location but I couldn't process it]", nil
	}

	name := getStringField(locationData, "name")
	if name == "" {
		name = "unnamed location"
	}
	address := getStringField(locationData, "address")

	var latitude, longitude float64
	if latVal, exists := locationData["latitude"]; exists {
		latitude, _ = latVal.(float64)
	}
	if longVal, exists := locationData["longitude"]; exists {
		longitude, _ = longVal.(float64)
	}

	locationInfo := fmt.Sprintf("[The user shared a location: \"%s\"", name)
	if address != "" {
		locationInfo += fmt.Sprintf(" (%s)", address)
	}
	locationInfo += fmt.Sprintf(" at coordinates: %f, %f]", latitude, longitude)

	return locationInfo, nil
}

// ContactHandler handles contact messages
type ContactHandler struct{}

// CanHandle checks if the message is a contact message
func (h *ContactHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "contactMessage"
}

// ProcessMessage processes a contact message
func (h *ContactHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a contact but I couldn't process it]", nil
	}
	contactData, exists := messageData["contactMessageData"].(map[string]interface{})
	if !exists {
		return "[The user sent a contact but I couldn't process it]", nil
	}

	displayName := getStringField(contactData, "displayName")
	if displayName == "" {
		displayName = "unknown contact"
	}
	vcard := getStringField(contactData, "vcard")

	phoneNumber := ""
	if vcard != "" {
		phoneMatch := strings.NewReader(vcard)
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(phoneMatch)
		vcardContent := buf.String()
		if strings.Contains(vcardContent, "TEL") {
			telParts := strings.Split(vcardContent, "TEL")
			if len(telParts) > 1 {
				colonParts := strings.Split(telParts[1], ":")
				if len(colonParts) > 1 {
					endParts := strings.Split(colonParts[1], "\n")
					if len(endParts) > 0 {
						phoneNumber = strings.TrimSpace(endParts[0])
					}
				}
			}
		}
	}

	contactInfo := fmt.Sprintf("[The user shared a contact: \"%s\"", displayName)
	if phoneNumber != "" {
		contactInfo += fmt.Sprintf(". Phone: %s", phoneNumber)
	}
	contactInfo += "]"

	return contactInfo, nil
}

// PollHandler handles incoming poll creation messages
type PollHandler struct{}

// CanHandle checks if the message is a poll creation message
func (h *PollHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "pollMessage"
}

// ProcessMessage processes a poll creation message
func (h *PollHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[Received poll message, but message data missing]", nil
	}
	pollData, exists := messageData["pollMessageData"].(map[string]interface{})
	if !exists {
		return "[Received poll message, but poll data missing]", nil
	}

	pollName := getStringField(pollData, "name")
	multipleAnswers := getBoolField(pollData, "multipleAnswers")
	optionsData := getSliceField(pollData, "options")

	var optionNames []string
	if optionsData != nil {
		for _, optionInterface := range optionsData {
			if optionMap, ok := optionInterface.(map[string]interface{}); ok {
				optionNames = append(optionNames, getStringField(optionMap, "optionName"))
			}
		}
	}

	pollInfo := fmt.Sprintf("[User sent a poll named \"%s\". Options: %s. Multiple answers allowed: %t]",
		pollName,
		strings.Join(optionNames, ", "),
		multipleAnswers)

	return pollInfo, nil
}

// PollUpdateHandler handles incoming poll update messages
type PollUpdateHandler struct{}

// CanHandle checks if the message is a poll update message
func (h *PollUpdateHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
	msgType, err := notification.MessageType()
	if err != nil {
		return false
	}
	return msgType == "pollUpdateMessage"
}

// ProcessMessage processes a poll update message
func (h *PollUpdateHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	messageData, exists := notification.Body["messageData"].(map[string]interface{})
	if !exists {
		return "[Received poll update, but message data missing]", nil
	}
	pollData, exists := messageData["pollMessageData"].(map[string]interface{})
	if !exists {
		return "[Received poll update, but poll data missing]", nil
	}

	pollName := getStringField(pollData, "name")
	multipleAnswers := getBoolField(pollData, "multipleAnswers")
	votesData := getSliceField(pollData, "votes")

	var voteSummaries []string
	if votesData != nil {
		for _, voteInterface := range votesData {
			if voteMap, ok := voteInterface.(map[string]interface{}); ok {
				optionName := getStringField(voteMap, "optionName")
				votersData := getSliceField(voteMap, "optionVoters")
				voterCount := 0
				if votersData != nil {
					voterCount = len(votersData)
				}
				voteSummaries = append(voteSummaries, fmt.Sprintf("\"%s\" (%d votes)", optionName, voterCount))
			}
		}
	}

	pollUpdateInfo := fmt.Sprintf("[User updated the poll named \"%s\". Current votes: %s. Multiple answers allowed: %t]",
		pollName,
		strings.Join(voteSummaries, ", "),
		multipleAnswers)

	return pollUpdateInfo, nil
}

// FallbackHandler handles unsupported message types
type FallbackHandler struct{}

// CanHandle always returns true as this is the fallback handler
func (h *FallbackHandler) CanHandle(_ *whatsapp_chatbot_golang.Notification) bool {
	return true
}

// ProcessMessage provides a fallback message for unsupported types
func (h *FallbackHandler) ProcessMessage(notification *whatsapp_chatbot_golang.Notification, _ *openai.Client, _ OpenAIModel) (interface{}, error) {
	msgType, err := notification.MessageType()
	if err != nil {
		msgType = "unknown"
	}
	log.Printf("FallbackHandler: Handling unknown or unhandled message type '%s'", msgType)
	return fmt.Sprintf("[The user sent a %s message that I can't process directly]", msgType), nil
}
