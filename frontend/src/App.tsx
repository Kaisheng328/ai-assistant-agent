import React, { useState, useEffect, useRef } from 'react'
import { Sidebar } from './components/Sidebar'
import { ChatArea } from './components/ChatArea'
import { KnowledgeBase } from './components/KnowledgeBase'
import { Conversation, Message, Settings, OllamaModel } from './types'
import { Sliders, X } from 'lucide-react'

const App: React.FC = () => {
  // Navigation & panels
  const [view, setView] = useState<'chat' | 'knowledge'>('chat')
  const [showSettingsModal, setShowSettingsModal] = useState(false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Domain states
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeConvId, setActiveConvId] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [settings, setSettings] = useState<Settings | null>(null)
  const [models, setModels] = useState<OllamaModel[]>([])

  // UI state
  const [isGenerating, setIsGenerating] = useState(false)
  const [streamStatus, setStreamStatus] = useState<string>('')
  const abortControllerRef = useRef<AbortController | null>(null)

  // Settings form states
  const [formSysPrompt, setFormSysPrompt] = useState('')
  const [formModel, setFormModel] = useState('')
  const [formEmbedModel, setFormEmbedModel] = useState('')
  const [formRagEnabled, setFormRagEnabled] = useState<boolean>(false)

  // Initialize data
  useEffect(() => {
    fetchSettings()
    fetchConversations()
    fetchOllamaModels()
  }, [])

  // Sync settings when fetched
  useEffect(() => {
    if (settings) {
      setFormSysPrompt(settings.system_prompt)
      setFormModel(settings.ollama_model)
      setFormEmbedModel(settings.ollama_embedding_model)
      setFormRagEnabled(settings.rag_enabled === 'true')
    }
  }, [settings])

  // Fetch all threads
  const fetchConversations = async () => {
    try {
      const res = await fetch('/api/conversations')
      if (!res.ok) throw new Error()
      const data = await res.json()
      setConversations(data || [])
    } catch (err) {
      console.error('Failed to fetch conversations')
    }
  }

  // Fetch settings
  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/settings')
      if (!res.ok) throw new Error()
      const data = await res.json()
      setSettings(data)
    } catch (err) {
      console.error('Failed to fetch settings')
    }
  }

  // Fetch Ollama models
  const fetchOllamaModels = async () => {
    try {
      const res = await fetch('/api/ollama/models')
      if (!res.ok) throw new Error()
      const data = await res.json()
      setModels(data.models || data || [])
    } catch (err) {
      console.error('Failed to fetch Ollama models. Local Ollama might not be running or ready.')
    }
  }

  // Handle new thread creation
  const handleNewConversation = async () => {
    try {
      const res = await fetch('/api/conversations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'New Conversation Thread' }),
      })
      if (!res.ok) throw new Error()
      const newConv = await res.json()
      
      // Update threads list
      await fetchConversations()
      
      // Switch view and set active
      setView('chat')
      setActiveConvId(newConv.id)
      setMessages([])
    } catch (err) {
      console.error('Failed to create new conversation thread')
    }
  }

  // Fetch message list for selected thread
  const handleSelectConversation = async (id: string) => {
    setActiveConvId(id)
    setView('chat')
    setIsGenerating(false)
    try {
      const res = await fetch(`/api/conversations/${id}/messages`)
      if (!res.ok) throw new Error()
      const data = await res.json()
      setMessages(data || [])
    } catch (err) {
      console.error('Failed to fetch messages')
    }
  }

  // Delete thread
  const handleDeleteConversation = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation() // Prevent selecting
    if (!window.confirm('Delete this conversation history?')) return

    try {
      const res = await fetch(`/api/conversations/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error()
      
      await fetchConversations()
      
      if (activeConvId === id) {
        setActiveConvId(null)
        setMessages([])
      }
    } catch (err) {
      console.error('Failed to delete conversation thread')
    }
  }

  // Send prompt & Stream response (Server-Sent Events)
  const handleSendMessage = async (content: string) => {
    if (!activeConvId || isGenerating) return

    setIsGenerating(true)
    setStreamStatus('')

    const controller = new AbortController()
    abortControllerRef.current = controller

    // 1. Create client-side temporary user message
    const tempUserMsg: Message = {
      id: 'temp-user-id',
      conversation_id: activeConvId,
      role: 'user',
      content: content,
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, tempUserMsg])

    // 2. Create client-side placeholder AI message
    const tempAiMsg: Message = {
      id: 'temp-ai-id',
      conversation_id: activeConvId,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, tempAiMsg])

    try {
      const res = await fetch(`/api/conversations/${activeConvId}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content }),
        signal: controller.signal,
      })

      if (!res.ok) {
        const errorText = await res.text()
        throw new Error(errorText || 'Server error')
      }

      const reader = res.body?.getReader()
      if (!reader) throw new Error('Response stream unavailable')

      const decoder = new TextDecoder('utf-8')
      let buffer = ''
      let accumulatedText = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')

        // Hold partial line in buffer
        buffer = lines.pop() || ''

        for (const line of lines) {
          const cleanedLine = line.trim()
          if (!cleanedLine.startsWith('data: ')) continue

          const jsonStr = cleanedLine.slice(6)
          try {
            const data = JSON.parse(jsonStr)

            // Sync original user message details once database commits it
            if (data.user_message) {
              setMessages((prev) =>
                prev.map((m) => (m.id === 'temp-user-id' ? data.user_message : m))
              )
            }

            if (data.status) {
              setStreamStatus(data.status)
            }

            if (data.token) {
              accumulatedText += data.token
              setMessages((prev) =>
                prev.map((m) => (m.id === 'temp-ai-id' ? { ...m, content: accumulatedText } : m))
              )
            }

            // Sync original AI response once complete and database commits it
            if (data.assistant_message) {
              setMessages((prev) =>
                prev.map((m) => (m.id === 'temp-ai-id' ? data.assistant_message : m))
              )
            }

            if (data.error) {
              accumulatedText += `\n[Error: ${data.error}]`
              setMessages((prev) =>
                prev.map((m) => (m.id === 'temp-ai-id' ? { ...m, content: accumulatedText } : m))
              )
            }
          } catch (e) {
            // Json parse fail (partial chunks, etc.)
          }
        }
      }
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === 'temp-ai-id'
              ? { ...m, content: m.content || 'Generation stopped.' }
              : m
          )
        )
        return
      }

      console.error('Streaming error: ', err)

      setMessages((prev) =>
        prev.map((m) =>
          m.id === 'temp-ai-id'
            ? { ...m, content: 'Reconnecting...' }
            : m
        )
      )

      for (let i = 0; i < 5; i++) {
        await new Promise(r => setTimeout(r, 3000))
        try {
          const recoverRes = await fetch(`/api/conversations/${activeConvId}/messages`)
          if (!recoverRes.ok) continue
          const recovered = await recoverRes.json()
          const last = recovered[recovered.length - 1]
          if (last && last.role === 'assistant' && last.content && !last.content.startsWith('Error:')) {
            setMessages(recovered)
            return
          }
        } catch {}
      }

      setMessages((prev) =>
        prev.map((m) =>
          m.id === 'temp-ai-id'
            ? { ...m, content: 'Connection lost. Please reload the page to see the response.' }
            : m
        )
      )
    } finally {
      setIsGenerating(false)
      setStreamStatus('')
      abortControllerRef.current = null
      fetchConversations()
    }
  }

  const handleStopGenerating = () => {
    abortControllerRef.current?.abort()
    setIsGenerating(false)
    setStreamStatus('')
  }

  // Handle settings update
  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          system_prompt: formSysPrompt,
          ollama_model: formModel,
          ollama_embedding_model: formEmbedModel,
          rag_enabled: formRagEnabled ? 'true' : 'false',
        }),
      })

      if (!res.ok) throw new Error()
      
      await fetchSettings()
      setShowSettingsModal(false)
    } catch (err) {
      alert('Failed to save settings')
    }
  }

  // Helper to get active thread title
  const getActiveConvTitle = () => {
    const matched = conversations.find((c) => c.id === activeConvId)
    return matched ? matched.title : 'Conversation'
  }

  return (
    <div className="app-container">
      {/* Sidebar history */}
      <Sidebar
        conversations={conversations}
        activeConvId={activeConvId}
        onSelectConv={handleSelectConversation}
        onNewConv={handleNewConversation}
        onDeleteConv={handleDeleteConversation}
        onOpenSettings={() => {
          fetchOllamaModels() // refresh models list
          setShowSettingsModal(true)
        }}
        isMobileOpen={isMobileSidebarOpen}
        onCloseMobile={() => setIsMobileSidebarOpen(false)}
      />

      {/* Main Panel View */}
      {view === 'chat' ? (
        <ChatArea
          activeConvId={activeConvId}
          activeConvTitle={getActiveConvTitle()}
          messages={messages}
          settings={settings}
          isGenerating={isGenerating}
          streamStatus={streamStatus}
          onSendMessage={handleSendMessage}
          onStopGenerating={handleStopGenerating}
          onOpenMobileSidebar={() => setIsMobileSidebarOpen(true)}
          onOpenKnowledge={() => setView('knowledge')}
        />
      ) : (
        <KnowledgeBase onClose={() => setView('chat')} />
      )}

      {/* Settings Modal */}
      {showSettingsModal && (
        <div className="modal-backdrop" onClick={() => setShowSettingsModal(false)}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <span className="modal-title">
                <Sliders size={18} className="text-indigo-400" />
                Studio Control Panel
              </span>
              <button className="modal-close" onClick={() => setShowSettingsModal(false)}>
                <X size={16} />
              </button>
            </div>

            <form onSubmit={handleSaveSettings}>
              <div className="modal-body">
                {/* System Prompt rule */}
                <div className="form-group">
                  <label className="form-label">System prompt rule</label>
                  <textarea
                    className="form-textarea"
                    placeholder="Instruct your Ollama AI on how to act, respond, and behave..."
                    value={formSysPrompt}
                    onChange={(e) => setFormSysPrompt(e.target.value)}
                    required
                  />
                </div>

                {/* Primary LLM Model */}
                <div className="form-group">
                  <label className="form-label">Generation Model (OpenRouter)</label>
                  <select
                    className="form-select"
                    value={formModel}
                    onChange={(e) => setFormModel(e.target.value)}
                  >
                    <option value="google/gemini-3.1-flash-lite">google/gemini-3.1-flash-lite</option>
                    <option value="deepseek/deepseek-v4-flash">deepseek/deepseek-v4-flash</option>
                  </select>
                </div>

                {/* Embedding model */}
                <div className="form-group">
                  <label className="form-label">Embedding Model (RAG Vectorizer)</label>
                  {models.length === 0 ? (
                    <input 
                      type="text" 
                      className="paste-input" 
                      value={formEmbedModel}
                      onChange={(e) => setFormEmbedModel(e.target.value)}
                      placeholder="Fallback embedding model name (e.g. nomic-embed-text)"
                    />
                  ) : (
                    <select
                      className="form-select"
                      value={formEmbedModel}
                      onChange={(e) => setFormEmbedModel(e.target.value)}
                    >
                      <option value="nomic-embed-text:latest">nomic-embed-text:latest (Default Embedding)</option>
                      {models.map((m) => (
                        <option key={m.name} value={m.name}>
                          {m.name} (Generate using LLM)
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {/* RAG Toggle switch */}
                <div className="toggle-group">
                  <div className="toggle-info">
                    <span className="toggle-title">Enable Semantic RAG Search</span>
                    <span className="toggle-desc">Retrieves relevant knowledge-base context prior to LLM query.</span>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={formRagEnabled}
                      onChange={(e) => setFormRagEnabled(e.target.checked)}
                    />
                    <span className="slider" />
                  </label>
                </div>
              </div>

              <div className="modal-footer">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setShowSettingsModal(false)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Apply & Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default App
