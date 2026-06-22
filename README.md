# GoAnsuran RAG Chatbot ("Ria")

AI-powered sales assistant for GoAnsuran (Mobile Wholesale City Malaysia). Answers questions about phone prices, installment plans, and eligibility using real product catalog data fetched live from the GoAnsuran API.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 18 + Vite + TypeScript |
| Backend | Go + Goyave v4 |
| Database | PostgreSQL 17 (GORM auto-migrate) |
| Vector DB | ChromaDB (L2 distance, 768-dim) |
| Embeddings | Ollama `nomic-embed-text` (local) |
| LLM | NVIDIA NIM `meta/llama-3.3-70b-instruct` |
| DB Inspector | Adminer |
| Orchestration | Docker Compose (6 services) |

---

## Architecture

```
Browser (localhost:3000)
  |
  |  React SPA (chat UI + knowledge base panel)
  |
  v
Goyave Backend (port 3000)
  |
  +---> PostgreSQL        Conversations, messages, settings
  +---> Ollama (11434)    Embedding generation (nomic-embed-text)
  +---> ChromaDB (8002)   Vector search for RAG retrieval
  +---> NVIDIA NIM API    LLM inference (llama-3.3-70b-instruct)
  +---> Adminer (8081)    Web-based DB inspector
```

### Message Flow

```
User sends message
  |
  v
1. POST /api/conversations/{id}/messages
2. Save user message to PostgreSQL
3. RAG retrieval:
   a. Ollama embeds the query (768-dim vector)
   b. ChromaDB returns top 10 nearest chunks (L2 distance <= 500)
   c. Matching chunks appended to system prompt as context
4. System prompt + history + context sent to NVIDIA NIM (stream=true)
5. Tokens streamed back to browser via SSE
6. Complete response saved to PostgreSQL
```

---

## Project Structure

```
chatbox-AI/
├── backend/
│   ├── main.go                          # Goyave entry point
│   ├── config.json                      # Server + database config
│   ├── go.mod
│   ├── database/
│   │   └── model/
│   │       ├── conversation.go          # Conversation model
│   │       ├── message.go               # Message model
│   │       └── setting.go               # Key-value settings model
│   └── http/
│       ├── controller/
│       │   ├── chat/chat.go             # Chat + RAG pipeline + SSE streaming
│       │   ├── knowledge/knowledge.go   # Knowledge CRUD + chunking + embedding
│       │   ├── ollama/ollama.go         # Model list
│       │   └── setting/setting.go       # Settings get/update
│       └── route/route.go               # API routes + SPA static serving
├── frontend/
│   ├── src/
│   │   ├── App.tsx                      # Main app + SSE handler
│   │   ├── main.tsx                     # React entry point
│   │   ├── types.ts                     # TypeScript interfaces
│   │   └── components/
│   │       ├── ChatArea.tsx             # Chat UI + markdown renderer
│   │       ├── Sidebar.tsx              # Conversation list
│   │       └── KnowledgeBase.tsx        # KB management panel
│   ├── index.html
│   ├── vite.config.ts
│   └── tsconfig.json
├── seed_goansuran.js                   # Main seeder (system prompt + 19 KB docs)
├── seed_catalog.js                     # Catalog seeder (100 live products from API)
├── docker-compose.yml                  # 6 services
└── .env                                # NVIDIA API URL + key
```

---

## API Endpoints

### Conversations

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/conversations` | List all conversations |
| `POST` | `/api/conversations` | Create new conversation |
| `DELETE` | `/api/conversations/{id}` | Delete conversation + messages |
| `GET` | `/api/conversations/{id}/messages` | Get all messages in conversation |
| `POST` | `/api/conversations/{id}/messages` | Send message + stream AI response |

**Send message request:**
```json
{ "content": "How much is iPhone 17 Pro Max 1TB?" }
```

**Send message response:** Server-Sent Events (SSE) stream:
```
data: {"user_message": {"id": "...", "role": "user", "content": "..."}}
data: {"status": "searching"}
data: {"status": "generating"}
data: {"token": "The"}
data: {"token": " iPhone"}
data: {"token": " 17 Pro Max"}
...
data: {"assistant_message": {"id": "...", "role": "assistant", "content": "..."}, "done": true}
```

### Settings

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/settings` | Get all settings (system_prompt, model, rag_enabled) |
| `POST` | `/api/settings` | Update settings |

**Update request:**
```json
{
  "system_prompt": "You are Ria...",
  "ollama_model": "meta/llama-3.3-70b-instruct",
  "ollama_embedding_model": "nomic-embed-text:latest",
  "rag_enabled": "true"
}
```

### Knowledge Base

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/knowledge` | List all indexed documents |
| `POST` | `/api/knowledge` | Upload + chunk + embed document |
| `DELETE` | `/api/knowledge` | Delete all documents |
| `DELETE` | `/api/knowledge/{id}` | Delete single document |

**Upload request:**
```json
{ "title": "Pricing: Apple iPhone", "content": "iPhone 17 Pro Max 1TB: RM 2,399 deposit..." }
```

### Ollama Models

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/ollama/models` | List available models |

---

## Setup

### 1. Prerequisites

- Docker + Docker Compose
- NVIDIA NIM API key (or compatible OpenAI-compatible endpoint)

### 2. Configure environment

Create `.env` in project root:

```env
ONLINE_API_URL=https://integrate.api.nvidia.com/v1
ONLINE_API_KEY=nvapi-your-key-here
```

### 3. Start all services

```bash
docker compose up -d
```

This starts 6 containers:
- `chatbox_frontend` — Builds the React app (one-shot)
- `chatbox_api_go` — Go backend on port 3000
- `chatbox_pgsql` — PostgreSQL database
- `ollama` — Local embedding model on port 11434
- `chromadb` — Vector database on port 8002
- `adminer` — DB inspector on port 8081

### 4. Pull embedding model

```bash
docker exec ollama ollama pull nomic-embed-text
```

### 5. Seed the knowledge base

```bash
# Seed system prompt + business knowledge (19 documents)
node seed_goansuran.js

# Seed live product catalog (100 products from api.ansuran2u.com)
node seed_catalog.js
```

### 6. Access the application

| URL | What |
|---|---|
| `http://localhost:3000` | Chat UI + Knowledge Base panel |
| `http://localhost:8081` | Adminer (DB: chatbox, User: chatbox, Pass: chatbox) |
| `http://localhost:8002` | ChromaDB API |
| `http://localhost:11434` | Ollama API |

---

## RAG Pipeline

### Knowledge Ingestion

```
Document text
  |
  v
chunkText(800 chars, 150 overlap)
  |
  v
Each chunk -> Ollama embedding -> 768-dim vector
  |
  v
ChromaDB stores: { id, embedding, text, metadata }
```

### Retrieval (per chat message)

```
User query
  |
  v
Ollama embedding (768-dim)
  |
  v
ChromaDB query (n_results=10, L2 distance)
  |
  v
Filter: distance <= 500 (relevance threshold)
  |
  v
Top chunks formatted with source title + relevance %
  |
  v
Appended to system prompt as context
```

### Seeded Knowledge (26 documents total)

**From `seed_goansuran.js` (19 docs):**
- Business overview, BNPL services, agent recruitment
- 4 installment plans: GoFlexi, GoAngkasa, JCL, BNPL
- Product catalog (17 categories: phones, motorcycles, appliances, etc.)
- Application process, eligibility, delivery/warranty policies
- Contact details, 7 branch locations
- FAQ: device condition, deposit policy, documents required, warranty
- Reviews & testimonials

**From `seed_catalog.js` (7 docs, 100 products):**
- Fetched live from `https://api.ansuran2u.com/v1/product/go-flexi`
- Grouped by brand: Apple iPhone, Apple iPad, Samsung, Honor, Oppo, Google
- Each product has exact deposit + monthly installment (GoFlexi 36-month plan)
- Price range summary: cheapest/most expensive per brand

---

## Configuration

### System Prompt

Configurable via Settings UI or `POST /api/settings`. Key rules:
- Responds in user's language (English/BM/Chinese)
- Quotes real prices from catalog context only
- Asks for model + storage before pricing (prices differ by storage)
- Uses markdown links `[text](url)` for all GoAnsuran URLs
- Never invents prices, models, or processes applications

### GoAnsuran URLs (used in chat responses)

| Page | URL |
|---|---|
| Main site | `https://goansuran.com/` |
| Sign-in/Register | `https://goansuran.com/auth/sign-in` |
| GoFlexi plan | `https://goansuran.com/go-flexi` |
| GoAngkasa plan | `https://goansuran.com/go-angkasa` |
| FAQ | `https://goansuran.com/faq` |
| Testimonials | `https://goansuran.com/testimonial` |
| WhatsApp | `https://wa.me/60196886440` |

### Docker Services

| Service | Image | Port | Volume |
|---|---|---|---|
| api | golang:latest | 3000 | `./backend:/app`, `./frontend/dist:/app/frontend/dist` |
| frontend | node:20-alpine | — | `./frontend:/app` |
| pgsql | postgres:17 | — | `db-data` (named volume) |
| ollama | ollama/ollama | 11434 | `ollama-data` (named volume) |
| chromadb | chromadb/chroma | 8002 | `chromadb-data` (named volume) |
| adminer | adminer | 8081 | — |

---

## Development

### Rebuild frontend after changes

```bash
docker compose restart frontend
```

### Restart backend after Go changes

```bash
docker compose restart api
```

### Re-seed knowledge base

```bash
node seed_goansuran.js   # Clears all docs, re-seeds system prompt + 19 docs
node seed_catalog.js     # Fetches 100 live products, seeds 7 pricing docs
```

### Access database via Adminer

Open `http://localhost:8081`:
- System: PostgreSQL
- Server: pgsql
- Username: chatbox
- Password: chatbox
- Database: chatbox

### Check ChromaDB state

```bash
# Count documents
COL_ID=$(curl -s -X POST http://localhost:8002/api/v2/tenants/default_tenant/databases/default_database/collections \
  -H "Content-Type: application/json" -d '{"name":"default_collection","get_or_create":true}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")

curl -s "http://localhost:8002/api/v2/tenants/default_tenant/databases/default_database/collections/$COL_ID/count"
```

### Check NVIDIA API connectivity

```bash
curl -s https://integrate.api.nvidia.com/v1/models \
  -H "Authorization: Bearer $ONLINE_API_KEY" | python3 -m json.tool
```

---

## Troubleshooting

| Issue | Fix |
|---|---|
| Chat shows "Error: Cloud API failed" | Check `.env` has valid NVIDIA API key |
| RAG not returning catalog data | Re-seed: `node seed_catalog.js` |
| Ollama embedding fails | Run: `docker exec ollama ollama pull nomic-embed-text` |
| Frontend not updating | Restart: `docker compose restart frontend` |
| Prices not matching | Re-run catalog seeder to fetch latest from GoAnsuran API |
| AI gives bare URLs (not clickable) | Re-seed: `node seed_goansuran.js` to update system prompt |
| Empty response in chat | Check API logs: `docker compose logs api --tail 20` |
