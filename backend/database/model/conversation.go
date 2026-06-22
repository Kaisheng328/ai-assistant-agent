package model

import (
	"time"
)

type Conversation struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	Messages  []Message `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}
