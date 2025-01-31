package main

import (
	"fmt"
	"os"
	"bytes"
	"log"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"gopkg.in/ini.v1"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func main() {
	// Read config.ini file
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Error loading config.ini: %v", err)
		os.Exit(1)
	}

	// Get Telegram bot token
	botToken := cfg.Section("telegram").Key("key").String()
	if botToken == "" {
		log.Fatalf("Can't find Telegram API key in config.ini")
		os.Exit(1)
	}

	// Initialize Telegram bot
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	updates, _ := bot.UpdatesViaLongPolling(nil)
	defer bot.StopLongPolling()

	// Process incoming messages
	for update := range updates {
		if update.Message != nil {
			chatID := tu.ID(update.Message.Chat.ID)

			textResponse, imageURL := GetGPTResponse(update.Message.Text)

			if imageURL != "" {
				// Send image to the user
				_, _ = bot.SendPhoto(tu.Photo(chatID, tu.FileFromURL(imageURL)))
			} else {
				// Send text response to the user
				_, _ = bot.SendMessage(tu.Message(chatID, textResponse))
			}
		}
	}
}

// GetGPTResponse handles user input and determines whether to generate text or an image
func GetGPTResponse(userInput string) (string, string) {
	if strings.HasPrefix(userInput, "Create an image:") {
		prompt := strings.TrimPrefix(userInput, "Create an image:")
		imageURL, err := GenerateImage(prompt)
		if err != nil {
			return "Error generating image", ""
		}
		return "Here is your image:", imageURL
	}

	// Read OpenAI API key from config.ini
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Error loading config.ini: %v", err)
		return "Error loading configuration", ""
	}

	apiKey := cfg.Section("api").Key("key").String()
	if apiKey == "" {
		log.Printf("API key not found in config.ini")
		return "API key not found", ""
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	// Create request body
	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "Be a helpful consultant, limit responses to 30 words, always reply in Ukrainian regardless of the input language"},
			{"role": "user", "content": userInput},
		},
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshalling request body: %v", err)
		return "Error processing request", ""
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "Error creating request", ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return "Error sending request", ""
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "Error reading response", ""
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: status code %d, body: %s", resp.StatusCode, string(body))
		return "API request error", ""
	}

	// Parse JSON response
	var chatResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &chatResponse); err != nil {
		log.Printf("Error unmarshalling response: %v", err)
		return "Error processing response", ""
	}

	if len(chatResponse.Choices) > 0 {
		return chatResponse.Choices[0].Message.Content, ""
	}

	log.Printf("No response from assistant")
	return "No response from AI", ""
}

// GenerateImage sends a request to OpenAI's image generation API
func GenerateImage(prompt string) (string, error) {
	// Read OpenAI API key from config.ini
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Error loading config.ini: %v", err)
		return "", err
	}

	apiKey := cfg.Section("api").Key("key").String()
	if apiKey == "" {
		log.Printf("API key not found in config.ini")
		return "", fmt.Errorf("API key not found")
	}

	apiURL := "https://api.openai.com/v1/images/generations"

	// Create request body
	requestBody := map[string]interface{}{
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshalling request body: %v", err)
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: status code %d, body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API request error")
	}

	// Parse JSON response
	var imageResponse struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &imageResponse); err != nil {
		log.Printf("Error unmarshalling response: %v", err)
		return "", err
	}

	if len(imageResponse.Data) > 0 {
		return imageResponse.Data[0].URL, nil
	}

	return "", fmt.Errorf("No image URL returned")
}