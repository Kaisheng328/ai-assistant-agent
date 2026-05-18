# Stage 1: Build the React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./frontend/
WORKDIR /app/frontend
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the Go backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
# We create a dummy go.sum or download dependencies with go mod tidy
COPY go.mod ./
COPY main.go ./
COPY backend/ ./backend/
# Copy the built frontend into the go embed path
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
# Run go mod tidy to generate go.sum inside build stage if not available
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 3: Run the application
FROM alpine:3.18
WORKDIR /workspace
COPY --from=backend-builder /app/main /usr/local/bin/chatbox_server
# Expose port 3000
EXPOSE 3000
# Environment variables
ENV PORT=3000
ENV DB_PATH=/app/db/chat.db
ENV OLLAMA_URL=http://ollama:11434
ENV CHROMA_URL=http://chromadb:8000
# Run the binary
CMD ["chatbox_server"]

