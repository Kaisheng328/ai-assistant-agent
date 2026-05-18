package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"chat_website/backend/enums"
	"chat_website/backend/models"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type DB struct {
	SQL *sql.DB
}

func InitDB() (*DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/db/chat.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %v", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %v", err)
	}

	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Printf("Warning: failed to set WAL mode: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		log.Printf("Warning: failed to enable foreign keys: %v", err)
	}

	db := &DB{SQL: sqlDB}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	if err := db.seedSettings(); err != nil {
		log.Printf("Warning: failed to seed default settings: %v", err)
	}

	return db, nil
}

func (db *DB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT CHECK(role IN ('user', 'assistant')) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if _, err := db.SQL.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) seedSettings() error {
	defaults := map[string]string{
		enums.SettingSystemPrompt:         "You are a helpful local AI assistant. Respond thoughtfully and concisely.",
		enums.SettingOllamaModel:          "llama3.2:3b",
		enums.SettingOllamaEmbeddingModel: "nomic-embed-text:latest",
		enums.SettingRagEnabled:           "false",
	}

	for k, v := range defaults {
		_, err := db.SQL.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.SQL.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.SQL.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (db *DB) GetConversations() ([]models.Conversation, error) {
	rows, err := db.SQL.Query(`SELECT id, title, created_at, updated_at FROM conversations ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var c models.Conversation
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&c.ID, &c.Title, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if c.CreatedAt.IsZero() {
			c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		c.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if c.UpdatedAt.IsZero() {
			c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}

func (db *DB) CreateConversation(title string) (*models.Conversation, error) {
	c := &models.Conversation{
		ID:        uuid.New().String(),
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.SQL.Exec(`INSERT INTO conversations (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		c.ID, c.Title, c.CreatedAt.Format("2006-01-02 15:04:05"), c.UpdatedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (db *DB) DeleteConversation(id string) error {
	_, err := db.SQL.Exec(`DELETE FROM conversations WHERE id = ?`, id)
	return err
}

func (db *DB) GetMessages(convID string) ([]models.Message, error) {
	rows, err := db.SQL.Query(`SELECT id, conversation_id, role, content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		var createdAtStr string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &createdAtStr); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if m.CreatedAt.IsZero() {
			m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (db *DB) AddMessage(convID, role, content string) (*models.Message, error) {
	m := &models.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now(),
	}

	tx, err := db.SQL.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO messages (id, conversation_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		m.ID, m.ConversationID, m.Role, m.Content, m.CreatedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`UPDATE conversations SET updated_at = ? WHERE id = ?`,
		m.CreatedAt.Format("2006-01-02 15:04:05"), m.ConversationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return m, nil
}
