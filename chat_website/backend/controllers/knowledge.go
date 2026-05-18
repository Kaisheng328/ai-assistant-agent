package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"chat_website/backend/services"
)

type KnowledgeController struct {
	Service *services.KnowledgeService
}

func NewKnowledgeController(svc *services.KnowledgeService) *KnowledgeController {
	return &KnowledgeController{
		Service: svc,
	}
}

func (h *KnowledgeController) RouteHandles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/knowledge")

	if path == "" || path == "/" {
		if r.Method == "GET" {
			h.ListDocuments(w, r)
		} else if r.Method == "POST" {
			h.UploadDocument(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && r.Method == "DELETE" {
		docID := parts[0]
		h.DeleteDocument(w, r, docID)
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *KnowledgeController) ListDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.Service.ListDocuments()
	if err != nil {
		http.Error(w, fmt.Sprintf("chromadb list error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func (h *KnowledgeController) UploadDocument(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	docID, totalChunks, err := h.Service.UploadDocument(body.Title, body.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"document_id": docID,
		"chunks":      totalChunks,
	})
}

func (h *KnowledgeController) DeleteDocument(w http.ResponseWriter, r *http.Request, docID string) {
	err := h.Service.DeleteDocument(docID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
