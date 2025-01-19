package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"github.com/go-ini/ini"
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
	// Читаем API ключ из файла config.ini
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Println("Error loading config.ini:", err)
		os.Exit(1)
	}

	apiKey := cfg.Section("api").Key("key").String()
	if apiKey == "" {
		fmt.Println("API key not found in config.ini")
		os.Exit(1)
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	// Создаем запрос
	requestBody := ChatGPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
				{Role: "system", Content: "Будь консультантом полезным, ответ не больше 30 слов"},
				{Role: "user", Content: "Сколько версий плейстешн существует?"},
		},
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling request body:", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: status code %d, body: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Парсим JSON-ответ
	var chatResponse ChatGPTResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		os.Exit(1)
	}

	// Выводим ответ ассистента
	if len(chatResponse.Choices) > 0 {
		fmt.Println("Assistant:", chatResponse.Choices[0].Message.Content)
	} else {
		fmt.Println("No response from assistant")
	}
}
