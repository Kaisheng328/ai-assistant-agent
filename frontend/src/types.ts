export interface Conversation {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface Settings {
  system_prompt: string;
  ollama_model: string;
  ollama_embedding_model: string;
  rag_enabled: 'true' | 'false';
}

export interface OllamaModel {
  name: string;
  modified_at: string;
  size: number;
}

export interface DocumentInfo {
  id: string;
  title: string;
  chunks: number;
  created_at: string;
}
