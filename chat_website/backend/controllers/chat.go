package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"chat_website/backend/enums"
	"chat_website/backend/models"
	"chat_website/backend/modules/chroma"
	"chat_website/backend/modules/db"
	"chat_website/backend/modules/ollama"
)

type ChatController struct {
	DB      *db.DB
	Ollama  *ollama.Client
	Chroma  *chroma.Client
	ColName string
}

func NewChatController(database *db.DB, ol *ollama.Client, chr *chroma.Client) *ChatController {
	return &ChatController{
		DB:      database,
		Ollama:  ol,
		Chroma:  chr,
		ColName: enums.DefaultCollectionName,
	}
}

func (h *ChatController) RouteHandles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/conversations")

	if path == "" || path == "/" {
		if r.Method == "GET" {
			h.GetConversations(w, r)
		} else if r.Method == "POST" {
			h.CreateConversation(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 {
		convID := parts[0]
		if r.Method == "DELETE" {
			h.DeleteConversation(w, r, convID)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "messages" {
		convID := parts[0]
		if r.Method == "GET" {
			h.GetMessages(w, r, convID)
		} else if r.Method == "POST" {
			h.SendMessageStream(w, r, convID)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *ChatController) GetConversations(w http.ResponseWriter, r *http.Request) {
	convs, err := h.DB.GetConversations()
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convs)
}

func (h *ChatController) CreateConversation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Title = "New Chat Thread"
	}
	if strings.TrimSpace(body.Title) == "" {
		body.Title = "New Chat Thread"
	}

	conv, err := h.DB.CreateConversation(body.Title)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create thread: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (h *ChatController) DeleteConversation(w http.ResponseWriter, r *http.Request, convID string) {
	err := h.DB.DeleteConversation(convID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete thread: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *ChatController) GetMessages(w http.ResponseWriter, r *http.Request, convID string) {
	msgs, err := h.DB.GetMessages(convID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get messages: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func (h *ChatController) SendMessageStream(w http.ResponseWriter, r *http.Request, convID string) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		http.Error(w, "invalid message content", http.StatusBadRequest)
		return
	}

	userMsg, err := h.DB.AddMessage(convID, enums.RoleUser, body.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to save message: %v", err), http.StatusInternalServerError)
		return
	}

	dbHistory, err := h.DB.GetMessages(convID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch thread history: %v", err), http.StatusInternalServerError)
		return
	}

	sysPrompt, _ := h.DB.GetSetting(enums.SettingSystemPrompt)
	modelName, _ := h.DB.GetSetting(enums.SettingOllamaModel)
	embedModelName, _ := h.DB.GetSetting(enums.SettingOllamaEmbeddingModel)
	ragEnabledStr, _ := h.DB.GetSetting(enums.SettingRagEnabled)
	ragEnabled := ragEnabledStr == "true"

	var ragContext string
	if ragEnabled {
		emb, err := h.Ollama.GetEmbeddings(embedModelName, body.Content)
		if err != nil {
			log.Printf("RAG Embedding retrieval failed: %v", err)
		} else {
			col, err := h.Chroma.GetOrCreateCollection(h.ColName)
			if err != nil {
				log.Printf("Chroma collection retrieve failed: %v", err)
			} else {
				res, err := h.Chroma.Query(col.ID, emb, 3)
				if err != nil {
					log.Printf("Chroma vector query failed: %v", err)
				} else if res != nil && len(res.Documents) > 0 && len(res.Documents[0]) > 0 {
					var contextBuilder strings.Builder
					contextBuilder.WriteString("\n=== RELEVANT CONTEXT FROM KNOWLEDGE BASE ===\n")
					for i, doc := range res.Documents[0] {
						contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, doc))
					}
					contextBuilder.WriteString("===========================================\n")
					ragContext = contextBuilder.String()
				}
			}
		}
	}

	var ollamaMsgs []models.ChatMessage

	fullSysPrompt := sysPrompt
	if ragContext != "" {
		fullSysPrompt = sysPrompt + "\nUse the following context to answer the user request. If the context doesn't contain relevant information or is insufficient, use your default training knowledge, but prioritize details in this context where applicable:\n" + ragContext
	}
	ollamaMsgs = append(ollamaMsgs, models.ChatMessage{
		Role:    "system",
		Content: fullSysPrompt,
	})

	for _, m := range dbHistory {
		ollamaMsgs = append(ollamaMsgs, models.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	initMeta, _ := json.Marshal(map[string]interface{}{
		"user_message": userMsg,
	})
	fmt.Fprintf(w, "data: %s\n\n", initMeta)
	flusher.Flush()

	var responseBuilder strings.Builder
	streamErr := h.Ollama.StreamChat(modelName, ollamaMsgs, func(token string) error {
		responseBuilder.WriteString(token)

		payload, _ := json.Marshal(map[string]string{
			"token": token,
		})

		_, err := fmt.Fprintf(w, "data: %s\n\n", payload)
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})

	if streamErr != nil {
		log.Printf("Streaming failed: %v", streamErr)
		errPayload, _ := json.Marshal(map[string]string{
			"error": fmt.Sprintf("Model inference error: %v", streamErr),
		})
		fmt.Fprintf(w, "data: %s\n\n", errPayload)
		flusher.Flush()
		return
	}

	aiResponse := responseBuilder.String()
	if strings.TrimSpace(aiResponse) != "" {
		aiMsg, dbErr := h.DB.AddMessage(convID, enums.RoleAssistant, aiResponse)
		if dbErr != nil {
			log.Printf("Error saving assistant message to DB: %v", dbErr)
		} else {
			finishPayload, _ := json.Marshal(map[string]interface{}{
				"assistant_message": aiMsg,
				"done":              true,
			})
			fmt.Fprintf(w, "data: %s\n\n", finishPayload)
			flusher.Flush()
		}
	}
}

func (h *ChatController) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "GET" {
		sysPrompt, _ := h.DB.GetSetting(enums.SettingSystemPrompt)
		modelName, _ := h.DB.GetSetting(enums.SettingOllamaModel)
		embedModel, _ := h.DB.GetSetting(enums.SettingOllamaEmbeddingModel)
		ragEnabled, _ := h.DB.GetSetting(enums.SettingRagEnabled)

		res := map[string]string{
			enums.SettingSystemPrompt:         sysPrompt,
			enums.SettingOllamaModel:          modelName,
			enums.SettingOllamaEmbeddingModel: embedModel,
			enums.SettingRagEnabled:           ragEnabled,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == "POST" {
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		for k, v := range updates {
			if k == enums.SettingSystemPrompt || k == enums.SettingOllamaModel || k == enums.SettingOllamaEmbeddingModel || k == enums.SettingRagEnabled {
				if err := h.DB.SetSetting(k, v); err != nil {
					http.Error(w, fmt.Sprintf("database write failure: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "settings updated"})
		return
	}

	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (h *ChatController) GetOllamaModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	models, err := h.Ollama.GetLocalModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("ollama connection error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
