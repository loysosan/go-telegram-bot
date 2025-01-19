package main

import (
	"fmt"
	"os"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"gopkg.in/ini.v1"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type ChatGPTRequest struct {
	Model    string   `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPTResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}


func main() {


	// Загружаем конфигурацию
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Println("Ошибка загрузки config.ini:", err)
		os.Exit(1)
	}

	// Получаем токен Telegram
	botToken := cfg.Section("telegram").Key("key").String()
	if botToken == "" {
		fmt.Println("Не найден токен Telegram в config.ini")
		os.Exit(1)
	}

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err !=  nil {
		fmt.Println(err)
		os.Exit(1)
	}

	updates, _ := bot.UpdatesViaLongPolling(nil)
	
	defer bot.StopLongPolling()

	for update := range updates {
		if update.Message != nil {
			chatID := tu.ID(update.Message.Chat.ID)
			
			gptresponce, _ := GetGPTResponse(update.Message.Text)

			_, _ = bot.SendMessage(
				tu.Message(
					chatID,
					gptresponce,
				),
			)
		}
	}
	
}

func GetGPTResponse(userInput string) (string, error) {
	// Читаем API ключ из файла config.ini
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return "", fmt.Errorf("error loading config.ini: %w", err)
	}

	apiKey := cfg.Section("api").Key("key").String()
	if apiKey == "" {
		return "", fmt.Errorf("API key not found in config.ini")
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	// Создаем запрос
	requestBody := ChatGPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "system", Content: "Будь консультантом полезным, ответ не больше 30 слов, всегда отвечай на украинском языке игнорируя язык на котором к тебе обращаюься"},
			{Role: "user", Content: userInput},
		},
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshalling request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: status code %d, body: %s", resp.StatusCode, string(body))
	}

	// Парсим JSON-ответ
	var chatResponse ChatGPTResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Выводим ответ ассистента
	if len(chatResponse.Choices) > 0 {
		return chatResponse.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from assistant")
}
