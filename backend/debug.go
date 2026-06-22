package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"name":          "default_collection",
		"get_or_create": true,
	})
	resp, err := http.Post("http://chromadb:8002/api/v2/tenants/default_tenant/databases/default_database/collections", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(b))
}
