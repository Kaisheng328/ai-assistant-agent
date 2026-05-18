package services

import (
	"fmt"
	"strings"
	"time"

	"chat_website/backend/enums"
	"chat_website/backend/models"
	"chat_website/backend/modules/chroma"
	"chat_website/backend/modules/db"
	"chat_website/backend/modules/ollama"

	"github.com/google/uuid"
)

type KnowledgeService struct {
	DB      *db.DB
	Ollama  *ollama.Client
	Chroma  *chroma.Client
	ColName string
}

func NewKnowledgeService(database *db.DB, ol *ollama.Client, chr *chroma.Client) *KnowledgeService {
	return &KnowledgeService{
		DB:      database,
		Ollama:  ol,
		Chroma:  chr,
		ColName: enums.DefaultCollectionName,
	}
}

func (s *KnowledgeService) ListDocuments() ([]models.DocumentInfo, error) {
	col, err := s.Chroma.GetOrCreateCollection(s.ColName)
	if err != nil {
		return nil, fmt.Errorf("chromadb error: %v", err)
	}

	result, err := s.Chroma.Get(col.ID, nil, 1000)
	if err != nil {
		return nil, fmt.Errorf("chromadb query error: %v", err)
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

	return docsList, nil
}

func (s *KnowledgeService) UploadDocument(title string, content string) (string, int, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if title == "" || content == "" {
		return "", 0, fmt.Errorf("title and content are required")
	}

	chunks := s.chunkText(content, 800, 150)
	if len(chunks) == 0 {
		return "", 0, fmt.Errorf("content is too short to index")
	}

	docID := uuid.New().String()
	embedModelName, _ := s.DB.GetSetting(enums.SettingOllamaEmbeddingModel)

	var ids []string
	var embeddings [][]float32
	var metadatas []map[string]interface{}

	for idx, chunk := range chunks {
		emb, err := s.Ollama.GetEmbeddings(embedModelName, chunk)
		if err != nil {
			return "", 0, fmt.Errorf("failed to generate embedding for chunk %d: %v", idx, err)
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
		ids = append(ids, chunkID)
		embeddings = append(embeddings, emb)
		metadatas = append(metadatas, map[string]interface{}{
			"document_id":  docID,
			"title":        title,
			"chunk_index":  idx,
			"total_chunks": len(chunks),
			"created_at":   time.Now().Format(time.RFC3339),
		})
	}

	col, err := s.Chroma.GetOrCreateCollection(s.ColName)
	if err != nil {
		return "", 0, fmt.Errorf("chromadb collection write failure: %v", err)
	}

	err = s.Chroma.AddEmbeddings(col.ID, ids, embeddings, metadatas, chunks)
	if err != nil {
		return "", 0, fmt.Errorf("failed to save embeddings to chromadb: %v", err)
	}

	return docID, len(chunks), nil
}

func (s *KnowledgeService) DeleteDocument(docID string) error {
	col, err := s.Chroma.GetOrCreateCollection(s.ColName)
	if err != nil {
		return fmt.Errorf("chromadb collection failure: %v", err)
	}

	err = s.Chroma.DeleteByMetadata(col.ID, map[string]interface{}{
		"document_id": docID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete document from chromadb: %v", err)
	}

	return nil
}

func (s *KnowledgeService) chunkText(text string, chunkSize int, overlap int) []string {
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
