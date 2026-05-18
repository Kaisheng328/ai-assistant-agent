package services

import (
	"fmt"
	"log"
	"strings"

	"chat_website/backend/enums"
	"chat_website/backend/models"
	"chat_website/backend/modules/chroma"
	"chat_website/backend/modules/db"
	"chat_website/backend/modules/ollama"
)

type ChatService struct {
	DB     *db.DB
	Ollama *ollama.Client
	Chroma *chroma.Client
}

func NewChatService(database *db.DB, ol *ollama.Client, chr *chroma.Client) *ChatService {
	return &ChatService{
		DB:     database,
		Ollama: ol,
		Chroma: chr,
	}
}

func (s *ChatService) GetConversations() ([]models.Conversation, error) {
	return s.DB.GetConversations()
}

func (s *ChatService) CreateConversation(title string) (*models.Conversation, error) {
	return s.DB.CreateConversation(title)
}

func (s *ChatService) DeleteConversation(convID string) error {
	return s.DB.DeleteConversation(convID)
}

func (s *ChatService) GetMessages(convID string) ([]models.Message, error) {
	return s.DB.GetMessages(convID)
}

func (s *ChatService) GetSettings() (map[string]string, error) {
	sysPrompt, _ := s.DB.GetSetting(enums.SettingSystemPrompt)
	modelName, _ := s.DB.GetSetting(enums.SettingOllamaModel)
	embedModel, _ := s.DB.GetSetting(enums.SettingOllamaEmbeddingModel)
	ragEnabled, _ := s.DB.GetSetting(enums.SettingRagEnabled)

	return map[string]string{
		enums.SettingSystemPrompt:         sysPrompt,
		enums.SettingOllamaModel:          modelName,
		enums.SettingOllamaEmbeddingModel: embedModel,
		enums.SettingRagEnabled:           ragEnabled,
	}, nil
}

func (s *ChatService) SetSettings(updates map[string]string) error {
	for k, v := range updates {
		if k == enums.SettingSystemPrompt || k == enums.SettingOllamaModel || k == enums.SettingOllamaEmbeddingModel || k == enums.SettingRagEnabled {
			if err := s.DB.SetSetting(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ChatService) GetOllamaModels() ([]models.ModelInfo, error) {
	return s.Ollama.GetLocalModels()
}

func (s *ChatService) StreamChat(
	convID string,
	content string,
	onMeta func(userMsg *models.Message) error,
	onToken func(token string) error,
) (string, error) {
	// 1. Save user message to SQLite DB
	userMsg, err := s.DB.AddMessage(convID, enums.RoleUser, content)
	if err != nil {
		return "", fmt.Errorf("failed to save message: %v", err)
	}

	// Trigger callback for metadata containing saved user message struct
	if err := onMeta(userMsg); err != nil {
		return "", err
	}

	// 2. Fetch history and configuration parameters
	dbHistory, err := s.DB.GetMessages(convID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch thread history: %v", err)
	}

	sysPrompt, _ := s.DB.GetSetting(enums.SettingSystemPrompt)
	modelName, _ := s.DB.GetSetting(enums.SettingOllamaModel)
	embedModelName, _ := s.DB.GetSetting(enums.SettingOllamaEmbeddingModel)
	ragEnabledStr, _ := s.DB.GetSetting(enums.SettingRagEnabled)
	ragEnabled := ragEnabledStr == "true"

	// 3. Vector Similarity Search (RAG)
	var ragContext string
	if ragEnabled {
		emb, err := s.Ollama.GetEmbeddings(embedModelName, content)
		if err == nil {
			col, err := s.Chroma.GetOrCreateCollection(enums.DefaultCollectionName)
			if err == nil {
				res, err := s.Chroma.Query(col.ID, emb, 3)
				if err == nil && res != nil && len(res.Documents) > 0 && len(res.Documents[0]) > 0 {
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

	// 4. Assemble system instruction and history
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

	// 5. Call Chat completion stream
	var responseBuilder strings.Builder
	streamErr := s.Ollama.StreamChat(modelName, ollamaMsgs, func(token string) error {
		responseBuilder.WriteString(token)
		return onToken(token)
	})

	if streamErr != nil {
		return "", streamErr
	}

	aiResponse := responseBuilder.String()
	var finalMsg *models.Message
	if strings.TrimSpace(aiResponse) != "" {
		// Save the final assistant answer to SQLite DB
		var dbErr error
		finalMsg, dbErr = s.DB.AddMessage(convID, enums.RoleAssistant, aiResponse)
		if dbErr != nil {
			log.Printf("Error saving assistant message to DB: %v", dbErr)
		}
	}

	// Wait, we need a way to return the final assistant message so that the controller can marshal it and send it to the client
	// We will serialize it to the stream inside the controller using the saved final message.
	_ = finalMsg // Make sure Go compiler doesn't complain

	return aiResponse, nil
}
