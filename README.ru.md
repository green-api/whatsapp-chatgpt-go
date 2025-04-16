# Библиотека WhatsApp GPT Bot Go

Современная библиотека для создания WhatsApp-бота с интеграцией OpenAI GPT, построенная на базе библиотеки GREEN-API для Golang.

## Особенности

- Интеграция с моделями OpenAI GPT для интеллектуальных ответов
- Поддержка различных моделей GPT (GPT-3.5, GPT-4, GPT-4o, O1)
- Мультимодальные возможности с поддержкой обработки изображений
- Транскрипция голосовых сообщений
- Комплексная обработка различных типов сообщений WhatsApp
- Архитектура промежуточного ПО (middleware) для настройки обработки сообщений и ответов
- Встроенное управление историей диалога

## Установка

```bash
go get github.com/green-api/whatsapp-chatgpt-go
```

Это также установит необходимые зависимости:
- `github.com/green-api/whatsapp-chatbot-golang`
- `github.com/sashabaranov/go-openai`

## Быстрый старт

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
    // Инициализация бота
    bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
        IDInstance:       "your-instance-id",
        APITokenInstance: "your-token",
        OpenAIApiKey:     "your-openai-api-key",
        Model:            whatsapp_chatgpt_go.ModelGPT4o,
        SystemMessage:    "Ты полезный ассистент.",
    })

    // Настройка обработки сигналов для корректного завершения
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Запуск бота в горутине
    go func() {
        log.Println("Запуск WhatsApp GPT бота...")
        bot.StartReceivingNotifications()
    }()

    // Ожидание сигнала завершения
    <-sigChan

    // Завершение работы
    log.Println("Завершение работы бота...")
    bot.StopReceivingNotifications()
    log.Println("Бот остановлен.")
}
```

## Варианты использования

Эта библиотека поддерживает два различных варианта использования в зависимости от ваших потребностей:

### 1. Самостоятельный бот

Вы можете запустить бота как самостоятельный сервис, который автоматически прослушивает и обрабатывает сообщения WhatsApp:

```go
bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    SystemMessage:    "Вы - полезный ассистент.",
})

// Начать прослушивание вебхуков и обработку сообщений
bot.StartReceivingNotifications()
```

### 2. Обработчик сообщений

Альтернативно, вы можете использовать бота как утилиту для обработки сообщений в вашем собственном приложении:

```go
gptBot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    SystemMessage:    "Вы - полезный ассистент.",
})

// Не нужно вызывать StartReceivingNotifications - просто используйте ProcessMessage когда нужно
response, updatedSessionData, err := gptBot.ProcessMessage(
    ctx, 
    notification,  // Уведомление из вашего обработчика вебхуков
    sessionData    // Ваши собственные данные сессии
)
if err != nil {
    // Обработка ошибки
}

// Обработайте ответ своим способом
// Сохраните обновленные данные сессии в своей системе состояний
```

## Основные компоненты

### Конфигурация бота

Полные параметры конфигурации для WhatsappGptBot:

```go
type GPTBotConfig struct {
    // API-ключ OpenAI
    OpenAIApiKey string
    
    // Модель для использования (по умолчанию: gpt-4o)
    Model OpenAIModel
    
    // Максимальное количество сообщений для хранения в истории (по умолчанию: 10)
    MaxHistoryLength int
    
    // Системное сообщение для установки поведения бота
    SystemMessage string
    
    // Температура для генерации ответов (по умолчанию: 0.5)
    Temperature float32
    
    // Сообщение об ошибке для отображения при возникновении проблем
    ErrorMessage string
    
    // ID инстанса GREEN-API
    IDInstance string
    
    // API токен инстанса GREEN-API
    APITokenInstance string
    
    // Следует ли очищать очередь вебхуков при запуске
    ClearWebhookQueueOnStart bool
}
```

### WhatsappGptBot

Основная структура для создания и управления вашим WhatsApp-ботом с поддержкой OpenAI:

```go
bot := whatsapp_chatgpt_go.NewWhatsappGptBot(whatsapp_chatgpt_go.GPTBotConfig{
    // Обязательные параметры
    IDInstance:       "your-instance-id",
    APITokenInstance: "your-token",
    OpenAIApiKey:     "your-openai-api-key",

    // Опциональные GPT-специфические параметры
    Model:            whatsapp_chatgpt_go.ModelGPT4o,
    MaxHistoryLength: 15,
    SystemMessage:    "Вы - полезный ассистент, специализирующийся на поддержке клиентов.",
    Temperature:      0.7,
    ErrorMessage:     "Извините, я не смог обработать ваш запрос. Пожалуйста, попробуйте снова.",
    
    // Опциональные параметры поведения
    ClearWebhookQueueOnStart: true,
})
```

## Обработка сообщений

Бот автоматически обрабатывает различные типы сообщений WhatsApp и преобразует их в формат, понятный моделям OpenAI.

### Поддерживаемые типы сообщений

- **Текст**
- **Изображения**
- **Аудио**
- **Видео**
- **Документы**
- **Опросы**
- **Геолокация**
- **Контакты**

### Реестр обработчиков сообщений

Бот использует реестр обработчиков сообщений для обработки различных типов сообщений:

```go
// Создание пользовательского обработчика сообщений
type CustomMessageHandler struct{}

func (h *CustomMessageHandler) CanHandle(notification *whatsapp_chatbot_golang.Notification) bool {
    // Логика для определения, может ли этот обработчик обработать сообщение
    return true
}

func (h *CustomMessageHandler) ProcessMessage(
    notification *whatsapp_chatbot_golang.Notification,
    client *openai.Client,
    model whatsapp_chatgpt_go.OpenAIModel,
) (interface{}, error) {
    // Обработка сообщения
    return "Обработанный контент", nil
}

// Регистрация пользовательского обработчика
bot.RegisterMessageHandler(&CustomMessageHandler{})
```

## Система промежуточного ПО (Middleware)

Система промежуточного ПО позволяет настраивать обработку сообщений перед отправкой в GPT и обработку ответов перед отправкой пользователю.

### Добавление промежуточного ПО для сообщений

```go
// Обработка сообщений перед отправкой в GPT
bot.AddMessageMiddleware(func(
    notification *whatsapp_chatbot_golang.Notification,
    messageContent interface{},
    messages []openai.ChatCompletionMessage,
    sessionData *whatsapp_chatgpt_go.GPTSessionData,
) (interface{}, []openai.ChatCompletionMessage, error) {
    // Добавление пользовательского контекста или изменение сообщения
    sender, _ := notification.Sender()
    log.Printf("Обработка сообщения от %s: %v", sender, messageContent)
    
    return messageContent, messages, nil
})
```

### Добавление промежуточного ПО для ответов

```go
// Обработка ответов GPT перед отправкой пользователю
bot.AddResponseMiddleware(func(
    response string,
    messages []openai.ChatCompletionMessage,
    sessionData *whatsapp_chatgpt_go.GPTSessionData,
) (string, []openai.ChatCompletionMessage, error) {
    // Форматирование или изменение ответа
    formattedResponse := response + "\n\n_Работает на GREEN-API_"
    
    return formattedResponse, messages, nil
})
```

## Данные сессии

GPT-бот расширяет базовые данные сессии:

```go
type GPTSessionData struct {
    // Сообщения в разговоре
    Messages []openai.ChatCompletionMessage `json:"messages"`
    
    // Временная метка последней активности
    LastActivity int64 `json:"lastActivity"`
    
    // Пользовательские данные
    UserData map[string]interface{} `json:"userData,omitempty"`
    
    // Контекст для текущего разговора
    Context map[string]interface{} `json:"context,omitempty"`
}
```

Вы можете получить доступ и изменить эти данные в вашем промежуточном ПО или через доступные методы.

## Поддерживаемые модели OpenAI

Библиотека поддерживает различные модели OpenAI:

### Модели GPT-4

- ModelGPT4 ("gpt-4")
- ModelGPT4Turbo ("gpt-4-turbo")
- ModelGPT4TurboPreview ("gpt-4-turbo-preview")
- ModelGPT41106Preview ("gpt-4-1106-preview")
- ModelGPT40125Preview ("gpt-4-0125-preview")
- ModelGPT432k ("gpt-4-32k")

### Модели GPT-4o

- ModelGPT4o ("gpt-4o") - по умолчанию
- ModelGPT4oMini ("gpt-4o-mini")

### Модели GPT-3.5

- ModelGPT35Turbo ("gpt-3.5-turbo")
- ModelGPT35Turbo16k ("gpt-3.5-turbo-16k")
- ModelGPT35Turbo1106 ("gpt-3.5-turbo-1106")
- ModelGPT35Turbo0125 ("gpt-3.5-turbo-0125")

### Модели o1

- ModelO1 ("o1")
- ModelO1Mini ("o1-mini")
- ModelO1Preview ("o1-preview")

### Модели с поддержкой изображений

Следующие модели могут обрабатывать изображения:

- ModelGPT4o ("gpt-4o")
- ModelGPT4oMini ("gpt-4o-mini")
- ModelGPT4Turbo ("gpt-4-turbo")

Вы можете проверить, поддерживает ли модель изображения, используя:

```go
if whatsapp_chatgpt_go.SupportsImages(bot.GetModel()) {
    // Обработка рабочего процесса на основе изображений
}
```

## Полный пример бота с промежуточным ПО

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
		log.Println("Предупреждение: Ошибка загрузки файла .env:", err)
	}

	idInstance := os.Getenv("GREEN_API_ID_INSTANCE")
	apiTokenInstance := os.Getenv("GREEN_API_TOKEN_INSTANCE")
	openaiToken := os.Getenv("OPENAI_API_KEY")

	if idInstance == "" || apiTokenInstance == "" || openaiToken == "" {
		log.Fatalf("Отсутствуют обязательные переменные окружения: GREEN_API_ID_INSTANCE, GREEN_API_TOKEN_INSTANCE, OPENAI_API_KEY")
	}

	config := whatsapp_chatgpt_go.GPTBotConfig{
		IDInstance:               idInstance,
		APITokenInstance:         apiTokenInstance,
		OpenAIApiKey:             openaiToken,
		Model:                    whatsapp_chatgpt_go.ModelGPT4o,
		MaxHistoryLength:         10,
		SystemMessage:            "Ты ассистент, отвечающий через WhatsApp.",
		Temperature:              0.7,
		ErrorMessage:             "Извините, я столкнулся с ошибкой при обработке вашего сообщения.",
		ClearWebhookQueueOnStart: true,
	}

	bot := whatsapp_chatgpt_go.NewWhatsappGptBot(config)

	// Пример промежуточного ПО: Логирует детали о входящих сообщениях.
	bot.AddMessageMiddleware(func(notification *whatsapp_chatbot_golang.Notification,
		messageContent interface{},
		messages []openai.ChatCompletionMessage,
		sessionData *whatsapp_chatgpt_go.GPTSessionData) (interface{}, []openai.ChatCompletionMessage, error) {

		sender, _ := notification.Sender()

		var contentLog string
		if parts, ok := messageContent.([]openai.ChatMessagePart); ok {
			contentLog = "Части мультиконтента: ["
			for i, p := range parts {
				if i > 0 {
					contentLog += ", "
				}
				contentLog += fmt.Sprintf("{Тип: %s, ", p.Type)
				if p.Type == openai.ChatMessagePartTypeText {
					contentLog += fmt.Sprintf("Текст: '%s'", p.Text)
				} else if p.Type == openai.ChatMessagePartTypeImageURL && p.ImageURL != nil {
					urlStr := p.ImageURL.URL
					if len(urlStr) > 50 {
						urlStr = urlStr[:47] + "..."
					}
					contentLog += fmt.Sprintf("URL изображения: %s", urlStr)
				} else {
					contentLog += "Другие данные"
				}
				contentLog += "}"
			}
			contentLog += "]"
		} else {
			contentLog = fmt.Sprintf("Текстовый контент: '%s'", truncateString(fmt.Sprintf("%v", messageContent), 100))
		}
		log.Printf("--> MID: Получено от %s: %s", sender, contentLog)
		log.Printf("--> MID: История содержит %d сообщений перед добавлением текущего.", len(messages))

		return messageContent, messages, nil
	})

	// Пример промежуточного ПО: Логирует отправляемый ответ.
	bot.AddResponseMiddleware(func(response string,
		messages []openai.ChatCompletionMessage,
		sessionData *whatsapp_chatgpt_go.GPTSessionData) (string, []openai.ChatCompletionMessage, error) {

		log.Printf("<-- MID: Отправка ответа: %s", truncateString(response, 100))
		log.Printf("<-- MID: История содержит %d сообщений после добавления ответа ассистента.", len(messages))
		return response, messages, nil
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Запуск WhatsApp GPT бота...")
		bot.StartReceivingNotifications()
		log.Println("Цикл получения уведомлений остановлен.")
	}()

	<-sigChan

	log.Println("Завершение работы бота...")
	bot.StopReceivingNotifications()
	log.Println("Бот остановлен.")
}
```

## Доступные методы

### WhatsappGptBot

- `NewWhatsappGptBot(config GPTBotConfig) *WhatsappGptBot` - Создает новый WhatsApp-бот с поддержкой GPT
- `StartReceivingNotifications()` - Начинает получение и обработку webhook-уведомлений
- `StopReceivingNotifications()` - Останавливает слушатель уведомлений
- `ProcessMessage(ctx context.Context, notification *whatsapp_chatbot_golang.Notification, sessionData *GPTSessionData) (string, *GPTSessionData, error)` - Обрабатывает сообщение без использования внутреннего менеджера состояний бота
- `AddMessageMiddleware(middleware ProcessMessageMiddleware)` - Регистрирует промежуточное ПО для обработки входящих сообщений
- `AddResponseMiddleware(middleware ProcessResponseMiddleware)` - Регистрирует промежуточное ПО для обработки ответов GPT
- `RegisterMessageHandler(handler MessageHandler)` - Добавляет пользовательский обработчик сообщений
- `GetOpenAI() *openai.Client` - Возвращает экземпляр клиента OpenAI
- `GetModel() OpenAIModel` - Возвращает идентификатор настроенной модели OpenAI
- `GetSystemMessage() string` - Возвращает настроенное системное сообщение
- `SupportsImages() bool` - Проверяет, поддерживает ли текущая настроенная модель обработку изображений
- `Methods()` - Доступ к методам базовой библиотеки для отправки сообщений и т.д.

## Лицензия

MIT