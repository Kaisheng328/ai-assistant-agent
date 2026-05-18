package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"chat_website/backend/models"
)

type Client struct {
	BaseURL      string
	HTTPClient   *http.Client
	OnlineAPIURL string
	OnlineAPIKey string
}

func NewClient() *Client {
	url := os.Getenv("OLLAMA_URL")
	if url == "" {
		url = "http://ollama:11434"
	}
	return &Client{
		BaseURL:      url,
		OnlineAPIURL: os.Getenv("ONLINE_API_URL"),
		OnlineAPIKey: os.Getenv("ONLINE_API_KEY"),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) GetEmbeddings(model string, prompt string) ([]float32, error) {
	// Always use local Ollama embedding models (e.g. nomic-embed-text) for high-speed offline vector processing
	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/embeddings", c.BaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama embeddings API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("ollama embeddings failed with status %d: %v", resp.StatusCode, errResp)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embeddings response: %v", err)
	}

	return result.Embedding, nil
}

func (c *Client) GetLocalModels() ([]models.ModelInfo, error) {
	var list []models.ModelInfo

	// If online provider is configured, prepend NVIDIA NIM models to the list
	if c.OnlineAPIKey != "" {
		nowStr := time.Now().Format(time.RFC3339)
		list = []models.ModelInfo{
			{Name: "nvidia/llama-3.1-nemotron-nano-8b-v1", Size: 0, ModifiedAt: nowStr},
			{Name: "meta/llama-3.1-8b-instruct", Size: 0, ModifiedAt: nowStr},
			{Name: "meta/llama-3.2-3b-instruct", Size: 0, ModifiedAt: nowStr},
			{Name: "google/gemma-3-12b-it", Size: 0, ModifiedAt: nowStr},
		}
	}

	locals, err := c.getLocalModelsOnly()
	if err == nil {
		list = append(list, locals...)
	}

	return list, nil
}

func (c *Client) getLocalModelsOnly() ([]models.ModelInfo, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/tags", c.BaseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama tags returned status %d", resp.StatusCode)
	}

	var result models.TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Models, nil
}

func (c *Client) PullModel(model string, onProgress func(status string, total, completed int64)) error {
	payload := map[string]interface{}{
		"name":   model,
		"stream": true,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/pull", c.BaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to call pull API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama pull API returned status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		var update struct {
			Status    string `json:"status"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
			Error     string `json:"error"`
		}

		if err := json.Unmarshal(line, &update); err == nil {
			if update.Error != "" {
				return fmt.Errorf("pull error: %s", update.Error)
			}
			onProgress(update.Status, update.Total, update.Completed)
		}
	}

	return nil
}

func (c *Client) StreamChat(model string, messages []models.ChatMessage, onToken func(token string) error) error {
	// Check if this model requires online routing (i.e. starts with nvidia/, meta/, google/ etc.)
	isOnline := c.OnlineAPIKey != "" && strings.Contains(model, "/")

	if isOnline {
		return c.streamOnlineChat(model, messages, onToken)
	}

	return c.streamLocalChat(model, messages, onToken)
}

func (c *Client) streamLocalChat(model string, messages []models.ChatMessage, onToken func(token string) error) error {
	payload := models.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/chat", c.BaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to establish chat stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("ollama chat returned status %d: %v", resp.StatusCode, errResp)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		var chunk models.ChatResponse
		if err := json.Unmarshal(line, &chunk); err == nil {
			if err := onToken(chunk.Message.Content); err != nil {
				return err
			}
			if chunk.Done {
				break
			}
		}
	}

	return nil
}

func (c *Client) streamOnlineChat(model string, messages []models.ChatMessage, onToken func(token string) error) error {
	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(c.OnlineAPIURL, "/"))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OnlineAPIKey))



	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call online chat API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("online chat API returned status %d: %v", resp.StatusCode, errResp)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 {
					token := chunk.Choices[0].Delta.Content
					if token != "" {
						if err := onToken(token); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}
