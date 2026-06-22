package model

import (
	"time"
)

type Message struct {
	ID             string       `gorm:"primaryKey;size:36" json:"id"`
	ConversationID string       `gorm:"size:36;index" json:"conversation_id"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"-"`
	Role           string       `gorm:"size:50;not null" json:"role"`
	Content        string       `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}
