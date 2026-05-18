package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"chat_website/backend/controllers"
	"chat_website/backend/modules/chroma"
	"chat_website/backend/modules/db"
	"chat_website/backend/modules/ollama"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

type spaHandler struct {
	fileServer http.Handler
	fs         fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api") {
		http.NotFound(w, r)
		return
	}

	cleanedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleanedPath == "" || cleanedPath == "." {
		cleanedPath = "index.html"
	}

	f, err := h.fs.Open(cleanedPath)
	if err != nil {
		indexFile, err := h.fs.Open("index.html")
		if err != nil {
			http.Error(w, "Frontend build files not found. Please compile frontend first.", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
		return
	}
	_ = f.Close()

	h.fileServer.ServeHTTP(w, r)
}

func main() {
	log.Println("Starting Chatbox AI System Backend (Modular Architecture)...")

	// 1. Initialize DB
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Fatal error initializing sqlite database: %v", err)
	}
	defer database.SQL.Close()
	log.Println("SQLite database module initialized successfully.")

	// 2. Initialize Ollama and ChromaDB Clients
	olClient := ollama.NewClient()
	chrClient := chroma.NewClient()

	log.Printf("Connecting to Ollama at: %s", olClient.BaseURL)
	log.Printf("Connecting to ChromaDB at: %s", chrClient.BaseURL)

	if chrClient.Heartbeat() {
		log.Println("ChromaDB connection successful.")
	} else {
		log.Println("Warning: ChromaDB is currently unreachable. Make sure ChromaDB container is running.")
	}

	// 3. Initialize Controllers
	chatController := controllers.NewChatController(database, olClient, chrClient)
	knowController := controllers.NewKnowledgeController(database, olClient, chrClient)

	// 4. Setup ServeMux
	mux := http.NewServeMux()

	// API Routing
	mux.HandleFunc("/api/conversations", chatController.RouteHandles)
	mux.HandleFunc("/api/conversations/", chatController.RouteHandles)
	mux.HandleFunc("/api/knowledge", knowController.RouteHandles)
	mux.HandleFunc("/api/knowledge/", knowController.RouteHandles)
	mux.HandleFunc("/api/settings", chatController.SettingsHandler)
	mux.HandleFunc("/api/ollama/models", chatController.GetOllamaModels)

	// Embedded Static Assets & React SPA Router
	subDist, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("Warning: Failed to locate embedded frontend files: %v", err)
		subDist = os.DirFS("./frontend/dist")
	}

	staticServer := http.FileServer(http.FS(subDist))
	mux.Handle("/", spaHandler{fileServer: staticServer, fs: subDist})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Application serving both API and Frontend on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server listen failed: %v", err)
	}
}
