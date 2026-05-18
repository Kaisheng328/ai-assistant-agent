package models

import "time"

// ==========================================
// Database Models
// ==========================================

// Conversation represents a chat thread
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents an individual message in SQLite
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // "user" or "assistant"
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// ==========================================
// Ollama Models
// ==========================================

// ChatMessage represents the schema for Ollama's chat API roles
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatRequest payload for Ollama chat API
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatResponse represents standard streaming response chunk from Ollama
type ChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
}

// ModelInfo represents a local Ollama model tag
type ModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

type TagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ==========================================
// ChromaDB Models
// ==========================================

// Collection represents a Chroma collection metadata
type Collection struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

// QueryResult represents standard search response
type QueryResult struct {
	IDs       [][]string                 `json:"ids"`
	Distances [][]float64                `json:"distances"`
	Metadatas [][]map[string]interface{} `json:"metadatas"`
	Documents [][]string                 `json:"documents"`
}

// GetResult represents standard direct retrieval response from /get
type GetResult struct {
	IDs       []string                 `json:"ids"`
	Metadatas []map[string]interface{} `json:"metadatas"`
	Documents []string                 `json:"documents"`
}

// ==========================================
// Knowledge / RAG Models
// ==========================================

// DocumentInfo represents grouped document chunks in memory
type DocumentInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Chunks    int       `json:"chunks"`
	CreatedAt time.Time `json:"created_at"`
}
