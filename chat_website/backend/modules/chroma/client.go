package chroma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"chat_website/backend/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	url := os.Getenv("CHROMA_URL")
	if url == "" {
		url = "http://chromadb:8000"
	}
	return &Client{
		BaseURL: url,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Heartbeat() bool {
	// Heartbeat is supported on api/v2 endpoints in newer ChromaDB versions
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/v2/heartbeat", c.BaseURL))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) GetOrCreateCollection(name string) (*models.Collection, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"name":          name,
		"get_or_create": true,
	})

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/v2/tenants/default_tenant/databases/default_database/collections", c.BaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call chroma collections API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("chromadb collections returned bad status: %d", resp.StatusCode)
	}

	var collection models.Collection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("failed to decode collection response: %v", err)
	}
	return &collection, nil
}

func (c *Client) AddEmbeddings(collectionID string, ids []string, embeddings [][]float32, metadatas []map[string]interface{}, documents []string) error {
	payload := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
		"documents":  documents,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/v2/tenants/default_tenant/databases/default_database/collections/%s/add", c.BaseURL, collectionID),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to call add embeddings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("chromadb add embeddings failed with status %d: %v", resp.StatusCode, errResp)
	}

	return nil
}

func (c *Client) Query(collectionID string, queryEmbedding []float32, nResults int) (*models.QueryResult, error) {
	payload := map[string]interface{}{
		"query_embeddings": [][]float32{queryEmbedding},
		"n_results":        nResults,
		"include":          []string{"documents", "metadatas", "distances"},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/v2/tenants/default_tenant/databases/default_database/collections/%s/query", c.BaseURL, collectionID),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query chromadb: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("chromadb query failed with status %d: %v", resp.StatusCode, errResp)
	}

	var result models.QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query result: %v", err)
	}

	return &result, nil
}

func (c *Client) DeleteByMetadata(collectionID string, filter map[string]interface{}) error {
	payload := map[string]interface{}{
		"where": filter,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v2/tenants/default_tenant/databases/default_database/collections/%s/delete", c.BaseURL, collectionID),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call chroma delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chromadb delete returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Get(collectionID string, ids []string, limit int) (*models.GetResult, error) {
	payload := map[string]interface{}{
		"include": []string{"metadatas", "documents"},
	}
	if len(ids) > 0 {
		payload["ids"] = ids
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/api/v2/tenants/default_tenant/databases/default_database/collections/%s/get", c.BaseURL, collectionID),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call chroma get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chromadb get returned status %d", resp.StatusCode)
	}

	var result models.GetResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode get result: %v", err)
	}

	return &result, nil
}
