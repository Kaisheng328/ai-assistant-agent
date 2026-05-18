package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"chat_website/backend/enums"
	"chat_website/backend/models"
	"chat_website/backend/services"
)

type ChatController struct {
	Service *services.ChatService
}

func NewChatController(svc *services.ChatService) *ChatController {
	return &ChatController{
		Service: svc,
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
	convs, err := h.Service.GetConversations()
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

	conv, err := h.Service.CreateConversation(body.Title)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create thread: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (h *ChatController) DeleteConversation(w http.ResponseWriter, r *http.Request, convID string) {
	err := h.Service.DeleteConversation(convID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete thread: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *ChatController) GetMessages(w http.ResponseWriter, r *http.Request, convID string) {
	msgs, err := h.Service.GetMessages(convID)
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Call the service StreamChat
	aiResponse, streamErr := h.Service.StreamChat(
		convID,
		body.Content,
		func(userMsg *models.Message) error {
			// Callback when metadata (user message) is ready
			initMeta, _ := json.Marshal(map[string]interface{}{
				"user_message": userMsg,
			})
			_, err := fmt.Fprintf(w, "data: %s\n\n", initMeta)
			if err != nil {
				return err
			}
			flusher.Flush()
			return nil
		},
		func(token string) error {
			// Callback on each stream token chunk
			payload, _ := json.Marshal(map[string]string{
				"token": token,
			})
			_, err := fmt.Fprintf(w, "data: %s\n\n", payload)
			if err != nil {
				return err
			}
			flusher.Flush()
			return nil
		},
	)

	if streamErr != nil {
		log.Printf("Streaming failed: %v", streamErr)
		errPayload, _ := json.Marshal(map[string]string{
			"error": fmt.Sprintf("Model inference error: %v", streamErr),
		})
		fmt.Fprintf(w, "data: %s\n\n", errPayload)
		flusher.Flush()
		return
	}

	// If streaming successfully finishes, send assistant message metadata
	if strings.TrimSpace(aiResponse) != "" {
		// Fetch assistant's message that was successfully saved in service layer
		msgs, _ := h.Service.GetMessages(convID)
		var assistantMsg *models.Message
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == enums.RoleAssistant {
				assistantMsg = &msgs[i]
				break
			}
		}

		if assistantMsg != nil {
			finishPayload, _ := json.Marshal(map[string]interface{}{
				"assistant_message": assistantMsg,
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
		settings, err := h.Service.GetSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get settings: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
		return
	}

	if r.Method == "POST" {
		var updates map[string]string
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := h.Service.SetSettings(updates); err != nil {
			http.Error(w, fmt.Sprintf("failed to save settings: %v", err), http.StatusInternalServerError)
			return
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

	models, err := h.Service.GetOllamaModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("ollama connection error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
