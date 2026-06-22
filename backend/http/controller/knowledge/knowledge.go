package knowledge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"api_go/database/model"
	"github.com/google/uuid"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
)

var chromaUrl = "http://chromadb:8000"
var ollamaUrl = "http://ollama:11434"

type ChromaDoc struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Chunks    int       `json:"chunks"`
	CreatedAt time.Time `json:"created_at"`
}

func getOrCreateCollection() (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"name":          "default_collection",
		"get_or_create": true,
	})
	resp, err := http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if id, ok := res["id"].(string); ok {
		return id, nil
	}
	return "", fmt.Errorf("id not found")
}

func Index(response *goyave.Response, request *goyave.Request) {
	colId, err := getOrCreateCollection()
	if err != nil {
		response.JSON(200, []interface{}{})
		return
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"limit":   1000,
		"include": []string{"metadatas"},
	})
	resp, err := http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/get", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		response.JSON(200, []interface{}{})
		return
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	docsMap := make(map[string]ChromaDoc)
	if metas, ok := res["metadatas"].([]interface{}); ok {
		for _, m := range metas {
			if meta, ok := m.(map[string]interface{}); ok {
				docId, _ := meta["document_id"].(string)
				if docId == "" {
					continue
				}
				if _, exists := docsMap[docId]; !exists {
					title, _ := meta["title"].(string)
					chunksFloat, _ := meta["total_chunks"].(float64)
					createdAtStr, _ := meta["created_at"].(string)
					createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
					docsMap[docId] = ChromaDoc{
						ID:        docId,
						Title:     title,
						Chunks:    int(chunksFloat),
						CreatedAt: createdAt,
					}
				}
			}
		}
	}

	out := make([]ChromaDoc, 0, len(docsMap))
	for _, doc := range docsMap {
		out = append(out, doc)
	}
	response.JSON(200, out)
}

func Upload(response *goyave.Response, request *goyave.Request) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	json.NewDecoder(request.Request().Body).Decode(&body)

	chunks := chunkText(body.Content, 800, 150)
	docId := uuid.New().String()

	var setting model.Setting
	database.Conn().Where("key = ?", "ollama_embedding_model").First(&setting)
	embedModel := "nomic-embed-text:latest"
	if setting.Value != "" {
		embedModel = setting.Value
	}

	var ids []string
	var embeddings [][]float64
	var metadatas []map[string]interface{}

	for idx, chunk := range chunks {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"model":  embedModel,
			"prompt": chunk,
		})
		resp, err := http.Post(ollamaUrl+"/api/embeddings", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			response.JSON(500, map[string]string{"error": "failed to embed"})
			return
		}
		var embRes map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&embRes)
		resp.Body.Close()

		if emb, ok := embRes["embedding"].([]interface{}); ok {
			var floatEmb []float64
			for _, e := range emb {
				floatEmb = append(floatEmb, e.(float64))
			}
			ids = append(ids, fmt.Sprintf("%s_chunk_%d", docId, idx))
			embeddings = append(embeddings, floatEmb)
			metadatas = append(metadatas, map[string]interface{}{
				"document_id":  docId,
				"title":        body.Title,
				"chunk_index":  idx,
				"total_chunks": len(chunks),
				"created_at":   time.Now().Format(time.RFC3339),
				"source":       "manual_upload",
			})
		}
	}

	colId, err := getOrCreateCollection()
	if err != nil {
		response.JSON(500, map[string]string{"error": "failed to create collection"})
		return
	}
	addBody, _ := json.Marshal(map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
		"documents":  chunks,
	})
	resp, err := http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/add", "application/json", bytes.NewBuffer(addBody))
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	} else {
		response.JSON(500, map[string]string{"error": "failed to save to chroma"})
		return
	}

	response.JSON(200, map[string]interface{}{
		"status":      "success",
		"document_id": docId,
		"chunks":      len(chunks),
	})
}

func Delete(response *goyave.Response, request *goyave.Request) {
	id := request.Params["id"]
	colId, _ := getOrCreateCollection()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"where": map[string]string{"document_id": id},
	})
	http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/delete", "application/json", bytes.NewBuffer(reqBody))
	response.JSON(200, map[string]string{"status": "deleted"})
}

func DeleteAll(response *goyave.Response, request *goyave.Request) {
	colId, err := getOrCreateCollection()
	if err != nil {
		response.JSON(200, map[string]interface{}{"status": "ok", "deleted": 0})
		return
	}

	getBody, _ := json.Marshal(map[string]interface{}{
		"limit":   10000,
		"include": []string{"metadatas"},
	})
	resp, err := http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/get", "application/json", bytes.NewBuffer(getBody))
	if err != nil {
		response.JSON(500, map[string]string{"error": "failed to fetch documents"})
		return
	}
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	resp.Body.Close()

	docIds := map[string]bool{}
	if metas, ok := res["metadatas"].([]interface{}); ok {
		for _, m := range metas {
			if meta, ok := m.(map[string]interface{}); ok {
				if docId, ok := meta["document_id"].(string); ok && docId != "" {
					docIds[docId] = true
				}
			}
		}
	}

	deleted := 0
	for docId := range docIds {
		delBody, _ := json.Marshal(map[string]interface{}{
			"where": map[string]string{"document_id": docId},
		})
		http.Post(chromaUrl+"/api/v2/tenants/default_tenant/databases/default_database/collections/"+colId+"/delete", "application/json", bytes.NewBuffer(delBody))
		deleted++
	}

	response.JSON(200, map[string]interface{}{"status": "all deleted", "deleted": deleted})
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
