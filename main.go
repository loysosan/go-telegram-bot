package main

import (
//	"fmt"
	"os"
	"bytes"
	"log"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"gopkg.in/ini.v1"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	
)

func main() {

	// Read config.ini file
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Error load config.ini:", err)
		os.Exit(1)
	}

	// Get Telegram tocken
	botToken := cfg.Section("telegram").Key("key").String()
	if botToken == "" {
		log.Fatalf("Cant find API key Telegram in config.ini")
		os.Exit(1)
	}

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err !=  nil {
		log.Println(err)
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
	// Read API OpenAI key form config.ini
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Error loading config.ini: %v", err)
		return "", nil
	}

	apiKey := cfg.Section("api").Key("key").String()
	if apiKey == "" {
		log.Printf("API key not found in config.ini")
		return "", nil
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	// Create request
	requestBody := ChatGPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "system", Content: "Будь консультантом полезным, ответ не больше 30 слов, всегда отвечай на украинском языке игнорируя язык на котором к тебе обращаюься"},
			{Role: "user", Content: userInput},
		},
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatalf("error marshalling request body: %w", err)
		return "", nil
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		log.Fatalf("error creating request: %w", err)
		return "", nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error sending request: %w", err)
		return "", nil
	}
	defer resp.Body.Close()

	// Read Answer
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading response body: %w", err)
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("error: status code %d, body: %s", resp.StatusCode, string(body))
		return "", nil
	}

	// Parse JSON-answer
	var chatResponse ChatGPTResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		log.Fatalf("error unmarshalling response: %w", err)
		return "", nil
	}

	// Return asistans answer
	if len(chatResponse.Choices) > 0 {
		return chatResponse.Choices[0].Message.Content, nil
	}
	log.Fatalf("no response from assistant")
	return "", nil
}
