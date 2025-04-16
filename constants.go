package whatsapp_chatgpt_go

// OpenAIModel represents an OpenAI model identifier
type OpenAIModel string

// OpenAI model constants
const (
	ModelGPT4             OpenAIModel = "gpt-4"
	ModelGPT4Turbo        OpenAIModel = "gpt-4-turbo"
	ModelGPT4TurboPreview OpenAIModel = "gpt-4-turbo-preview"
	ModelGPT41106Preview  OpenAIModel = "gpt-4-1106-preview"
	ModelGPT40125Preview  OpenAIModel = "gpt-4-0125-preview"
	ModelGPT432k          OpenAIModel = "gpt-4-32k"
	ModelGPT4o            OpenAIModel = "gpt-4o"
	ModelGPT4oMini        OpenAIModel = "gpt-4o-mini"
	ModelGPT35Turbo       OpenAIModel = "gpt-3.5-turbo"
	ModelGPT35Turbo16k    OpenAIModel = "gpt-3.5-turbo-16k"
	ModelGPT35Turbo1106   OpenAIModel = "gpt-3.5-turbo-1106"
	ModelGPT35Turbo0125   OpenAIModel = "gpt-3.5-turbo-0125"
	ModelO1               OpenAIModel = "o1"
	ModelO1Mini           OpenAIModel = "o1-mini"
	ModelO1Preview        OpenAIModel = "o1-preview"
	DefaultModel                      = ModelGPT4o
)

// ImageCapableModels lists models known to support image input.
var ImageCapableModels = map[OpenAIModel]bool{
	ModelGPT4o:     true,
	ModelGPT4oMini: true,
	ModelGPT4Turbo: true,
}

// SupportsImages checks if a model supports image processing based on the known list.
func SupportsImages(model OpenAIModel) bool {
	_, supported := ImageCapableModels[model]
	return supported
}

// String returns the string representation of the OpenAIModel.
func (m OpenAIModel) String() string {
	return string(m)
}
