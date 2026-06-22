package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"api_go/database/model"

	"github.com/google/uuid"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func writeSSE(response *goyave.Response, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(response, "data: %s\n\n", string(jsonBytes))
	if f, ok := response.Writer().(http.Flusher); ok {
		f.Flush()
	}
}

func heartbeat(response *goyave.Response) {
	fmt.Fprintf(response, ": keep-alive\n\n")
	if f, ok := response.Writer().(http.Flusher); ok {
		f.Flush()
	}
}

func Index(response *goyave.Response, request *goyave.Request) {
	var convs []model.Conversation
	database.Conn().Order("updated_at desc").Find(&convs)
	response.JSON(200, convs)
}

func Create(response *goyave.Response, request *goyave.Request) {
	var body struct {
		Title string `json:"title"`
	}
	json.NewDecoder(request.Request().Body).Decode(&body)
	if body.Title == "" {
		body.Title = "New Chat Thread"
	}

	conv := model.Conversation{
		ID:    uuid.New().String(),
		Title: body.Title,
	}
	database.Conn().Create(&conv)
	response.JSON(200, conv)
}

func Delete(response *goyave.Response, request *goyave.Request) {
	id := request.Params["id"]
	database.Conn().Where("id = ?", id).Delete(&model.Conversation{})
	response.JSON(200, map[string]string{"status": "deleted"})
}

func Messages(response *goyave.Response, request *goyave.Request) {
	id := request.Params["id"]
	var msgs []model.Message
	database.Conn().Where("conversation_id = ?", id).Order("created_at asc").Find(&msgs)
	response.JSON(200, msgs)
}

func SendMessage(response *goyave.Response, request *goyave.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(request.Request().Body).Decode(&body); err != nil || body.Content == "" {
		response.JSON(400, map[string]string{"error": "invalid content"})
		return
	}

	convID := request.Params["id"]

	userMsg := model.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           "user",
		Content:        body.Content,
	}
	database.Conn().Create(&userMsg)
	database.Conn().Model(&model.Conversation{}).Where("id = ?", convID).Update("updated_at", time.Now())

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")

	response.WriteHeader(http.StatusOK)

	writeSSE(response, map[string]interface{}{"user_message": userMsg})

	// Settings
	var sysPromptSetting model.Setting
	database.Conn().Where("key = ?", "system_prompt").First(&sysPromptSetting)
	sysPrompt := "You are a helpful AI assistant."
	if sysPromptSetting.Value != "" {
		sysPrompt = sysPromptSetting.Value
	}

	var modelSetting model.Setting
	database.Conn().Where("key = ?", "ollama_model").First(&modelSetting)
	modelName := "meta/llama-3.2-3b-instruct"
	if modelSetting.Value != "" {
		modelName = modelSetting.Value
	}

	// Fetch history
	var history []model.Message
	database.Conn().Where("conversation_id = ?", convID).Order("created_at asc").Find(&history)

	var ragSetting model.Setting
	database.Conn().Where("key = ?", "rag_enabled").First(&ragSetting)
	
	if ragSetting.Value == "true" {
		heartbeat(response)
		writeSSE(response, map[string]string{"status": "searching"})
		var embedSetting model.Setting
		database.Conn().Where("key = ?", "ollama_embedding_model").First(&embedSetting)
		embedModel := "nomic-embed-text:latest"
		if embedSetting.Value != "" { embedModel = embedSetting.Value }
		
		reqBody, _ := json.Marshal(map[string]interface{}{
			"model": embedModel,
			"prompt": body.Content,
		})
		resp, _ := http.Post("http://ollama:11434/api/embeddings", "application/json", bytes.NewBuffer(reqBody))
		var embRes map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&embRes)
		resp.Body.Close()
		
		if emb, ok := embRes["embedding"].([]interface{}); ok {
			var floatEmb []float64
			for _, e := range emb { floatEmb = append(floatEmb, e.(float64)) }
			
			// get collection id
			getColBody, _ := json.Marshal(map[string]interface{}{"name": "default_collection", "get_or_create": true})
			colResp, _ := http.Post("http://chromadb:8000/api/v2/tenants/default_tenant/databases/default_database/collections", "application/json", bytes.NewBuffer(getColBody))
			var colRes map[string]interface{}
			json.NewDecoder(colResp.Body).Decode(&colRes)
			colResp.Body.Close()
			colId := colRes["id"].(string)
			
		queryBody, _ := json.Marshal(map[string]interface{}{
			"query_embeddings": [][]float64{floatEmb},
			"n_results":        10,
			"include":          []string{"documents", "metadatas", "distances"},
		})
			queryResp, _ := http.Post("http://chromadb:8000/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/query", "application/json", bytes.NewBuffer(queryBody))
			var queryRes map[string]interface{}
			json.NewDecoder(queryResp.Body).Decode(&queryRes)
			queryResp.Body.Close()
			
		if docs, ok := queryRes["documents"].([]interface{}); ok && len(docs) > 0 {
			if chunkArr, ok := docs[0].([]interface{}); ok && len(chunkArr) > 0 {
				// Extract distances and metadatas for filtering
				var distances []interface{}
				if distRaw, ok := queryRes["distances"].([]interface{}); ok && len(distRaw) > 0 {
					distances, _ = distRaw[0].([]interface{})
				}
				var metadatas []interface{}
				if metaRaw, ok := queryRes["metadatas"].([]interface{}); ok && len(metaRaw) > 0 {
					metadatas, _ = metaRaw[0].([]interface{})
				}

				var contextBuilder strings.Builder
				contextBuilder.WriteString("\n\n=== RELEVANT CONTEXT FROM KNOWLEDGE BASE ===\n")

				relevantCount := 0
				for i, c := range chunkArr {
					docText, _ := c.(string)

					// Distance-based relevance filtering (L2 distance with nomic-embed-text 768-dim)
					// Typical relevant range: 280-450; irrelevant: 500+
					var distance float64 = 999.0
					if i < len(distances) {
						distance, _ = distances[i].(float64)
					}
					if distance > 500.0 {
						continue
					}

					// Extract source title
					sourceTitle := "Unknown"
					if i < len(metadatas) {
						if meta, ok := metadatas[i].(map[string]interface{}); ok {
							if title, ok := meta["title"].(string); ok {
								sourceTitle = title
							}
						}
					}

					// Calculate relevance percentage (L2 scale: 0=perfect, 500=irrelevant)
					relevance := int((1.0 - distance/500.0) * 100)
					if relevance < 0 {
						relevance = 0
					}

					relevantCount++
					contextBuilder.WriteString(fmt.Sprintf("[%d] (Source: %s, Relevance: %d%%)\n%s\n\n", relevantCount, sourceTitle, relevance, docText))
				}

				if relevantCount > 0 {
					contextBuilder.WriteString("===========================================\n")
					sysPrompt += contextBuilder.String()
				}
			}
		}
		}
	}

	var reqMsgs []ChatMessage
	reqMsgs = append(reqMsgs, ChatMessage{Role: "system", Content: sysPrompt})
	for _, m := range history {
		reqMsgs = append(reqMsgs, ChatMessage{Role: m.Role, Content: m.Content})
	}

	apiUrl := os.Getenv("ONLINE_API_URL")
	if apiUrl == "" {
		apiUrl = "https://integrate.api.nvidia.com/v1"
	}
	apiKey := os.Getenv("ONLINE_API_KEY")

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":       modelName,
		"messages":    reqMsgs,
		"stream":      true,
		"max_tokens":  2048,
	})

	fmt.Println("SENDING TO NVIDIA:", string(reqBody))
	heartbeat(response)
	writeSSE(response, map[string]string{"status": "generating"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:     false,
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", apiUrl+"/chat/completions", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	res, err := client.Do(req)
	var fullContent string
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fullContent = "I'm taking longer than expected to respond. Please try again or visit [goansuran.com](https://goansuran.com/)."
		} else {
			fullContent = "Error: Cloud API connection failed. Please try again."
		}
		writeSSE(response, map[string]string{"token": fullContent})
	} else {
		defer res.Body.Close()
		scanner := bufio.NewScanner(res.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 6 && line[:6] == "data: " {
				dataStr := line[6:]
				if dataStr == "[DONE]" {
					break
				}
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
					if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
						if choice, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := choice["delta"].(map[string]interface{}); ok {
								if content, ok := delta["content"].(string); ok && content != "" {
									fullContent += content
									writeSSE(response, map[string]string{"token": content})
								}
							}
						}
					}
				}
			}
		}
		if ctx.Err() == context.DeadlineExceeded && fullContent == "" {
			fullContent = "I'm taking longer than expected to respond. Please try again or visit [goansuran.com](https://goansuran.com/)."
			writeSSE(response, map[string]string{"token": fullContent})
		}
	}

	assistantMsg := model.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           "assistant",
		Content:        fullContent,
	}
	database.Conn().Create(&assistantMsg)

	writeSSE(response, map[string]interface{}{"assistant_message": assistantMsg, "done": true})
}
