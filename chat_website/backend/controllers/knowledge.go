package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"chat_website/backend/enums"
	"chat_website/backend/models"
	"chat_website/backend/modules/chroma"
	"chat_website/backend/modules/db"
	"chat_website/backend/modules/ollama"

	"github.com/google/uuid"
)

type KnowledgeController struct {
	DB      *db.DB
	Ollama  *ollama.Client
	Chroma  *chroma.Client
	ColName string
}

func NewKnowledgeController(database *db.DB, ol *ollama.Client, chr *chroma.Client) *KnowledgeController {
	return &KnowledgeController{
		DB:      database,
		Ollama:  ol,
		Chroma:  chr,
		ColName: enums.DefaultCollectionName,
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
	col, err := h.Chroma.GetOrCreateCollection(h.ColName)
	if err != nil {
		http.Error(w, fmt.Sprintf("chromadb error: %v", err), http.StatusInternalServerError)
		return
	}

	result, err := h.Chroma.Get(col.ID, nil, 1000)
	if err != nil {
		http.Error(w, fmt.Sprintf("chromadb query error: %v", err), http.StatusInternalServerError)
		return
	}

	docsMap := make(map[string]*models.DocumentInfo)

	if result != nil && len(result.Metadatas) > 0 {
		for _, item := range result.Metadatas {
			if item == nil {
				continue
			}
			docIDRaw, ok1 := item["document_id"]
			titleRaw, ok2 := item["title"]
			if !ok1 || !ok2 {
				continue
			}
			docID := fmt.Sprintf("%v", docIDRaw)
			title := fmt.Sprintf("%v", titleRaw)

			totalChunks := 1
			if tot, ok := item["total_chunks"]; ok {
				if val, okNum := tot.(float64); okNum {
					totalChunks = int(val)
				} else if valInt, okInt := tot.(int); okInt {
					totalChunks = valInt
				}
			}

			createdAt := time.Now()
			if cat, ok := item["created_at"]; ok {
				if t, errParse := time.Parse(time.RFC3339, fmt.Sprintf("%v", cat)); errParse == nil {
					createdAt = t
				}
			}

			if _, exists := docsMap[docID]; !exists {
				docsMap[docID] = &models.DocumentInfo{
					ID:        docID,
					Title:     title,
					Chunks:    totalChunks,
					CreatedAt: createdAt,
				}
			}
		}
	}

	docsList := make([]models.DocumentInfo, 0, len(docsMap))
	for _, doc := range docsMap {
		docsList = append(docsList, *doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docsList)
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

	body.Title = strings.TrimSpace(body.Title)
	body.Content = strings.TrimSpace(body.Content)

	if body.Title == "" || body.Content == "" {
		http.Error(w, "title and content are required fields", http.StatusBadRequest)
		return
	}

	chunks := chunkText(body.Content, 800, 150)
	if len(chunks) == 0 {
		http.Error(w, "content is too short to index", http.StatusBadRequest)
		return
	}

	docID := uuid.New().String()
	embedModelName, _ := h.DB.GetSetting(enums.SettingOllamaEmbeddingModel)

	var ids []string
	var embeddings [][]float32
	var metadatas []map[string]interface{}

	for idx, chunk := range chunks {
		emb, err := h.Ollama.GetEmbeddings(embedModelName, chunk)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate embedding for chunk %d: %v", idx, err), http.StatusInternalServerError)
			return
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
		ids = append(ids, chunkID)
		embeddings = append(embeddings, emb)
		metadatas = append(metadatas, map[string]interface{}{
			"document_id":  docID,
			"title":        body.Title,
			"chunk_index":  idx,
			"total_chunks": len(chunks),
			"created_at":   time.Now().Format(time.RFC3339),
		})
	}

	col, err := h.Chroma.GetOrCreateCollection(h.ColName)
	if err != nil {
		http.Error(w, fmt.Sprintf("chromadb collection write failure: %v", err), http.StatusInternalServerError)
		return
	}

	err = h.Chroma.AddEmbeddings(col.ID, ids, embeddings, metadatas, chunks)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to save embeddings to chromadb: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"document_id": docID,
		"chunks":      len(chunks),
	})
}

func (h *KnowledgeController) DeleteDocument(w http.ResponseWriter, r *http.Request, docID string) {
	col, err := h.Chroma.GetOrCreateCollection(h.ColName)
	if err != nil {
		http.Error(w, fmt.Sprintf("chromadb collection failure: %v", err), http.StatusInternalServerError)
		return
	}

	err = h.Chroma.DeleteByMetadata(col.ID, map[string]interface{}{
		"document_id": docID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete document from chromadb: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func chunkText(text string, chunkSize int, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	textLength := len(runes)

	if textLength == 0 {
		return chunks
	}

	if textLength <= chunkSize {
		return []string{text}
	}

	for i := 0; i < textLength; {
		end := i + chunkSize
		if end > textLength {
			end = textLength
		}

		chunks = append(chunks, string(runes[i:end]))

		if end == textLength {
			break
		}

		i += chunkSize - overlap
		if i >= textLength {
			break
		}
	}

	return chunks
}
