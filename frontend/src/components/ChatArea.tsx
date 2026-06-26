import React, { useState, useRef, useEffect } from 'react'
import { Send, Menu, Sparkles, BookOpen, Check, Copy, Square } from 'lucide-react'
import { Message, Settings } from '../types'

interface ChatAreaProps {
  activeConvId: string | null;
  activeConvTitle: string;
  messages: Message[];
  settings: Settings | null;
  isGenerating: boolean;
  streamStatus: string;
  onSendMessage: (content: string) => void;
  onStopGenerating: () => void;
  onOpenMobileSidebar: () => void;
  onOpenKnowledge: () => void;
}

// Custom Premium Code Block Renderer with copy functionality
const CodeBlock: React.FC<{ code: string; language: string }> = ({ code, language }) => {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy text', err)
    }
  }

  return (
    <div className="code-block-card">
      <div className="code-block-header">
        <span className="code-block-lang">{language || 'code'}</span>
        <button className="code-block-copy-btn" onClick={handleCopy}>
          {copied ? (
            <>
              <Check size={12} className="text-emerald-400" />
              <span className="text-emerald-400">Copied!</span>
            </>
          ) : (
            <>
              <Copy size={12} />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>
      <pre className="code-block-pre">
        <code>{code.trim()}</code>
      </pre>
    </div>
  )
}

// Custom Markdown parsing engine (completely Vanilla React)
const MarkdownRenderer: React.FC<{ text: string }> = ({ text }) => {
  if (!text) return null

  const escapeHTML = (str: string) => {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;')
  }

  const formatInline = (html: string) => {
    return html
      .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      .replace(/`(.*?)`/g, '<code class="inline-code">$1</code>')
  }

  // 1. Split into code blocks vs non-code blocks
  const parts = text.split(/```/)
  return (
    <div className="markdown-container">
      {parts.map((part, index) => {
        // Odd indexes are code blocks
        if (index % 2 !== 0) {
          const lines = part.split('\n')
          const language = lines[0] ? lines[0].trim() : ''
          const code = lines.slice(1).join('\n')
          return <CodeBlock key={index} code={code} language={language} />
        }

        // Even indexes are standard markdown text
        const textLines = part.split('\n')
        return (
          <div key={index} style={{ display: 'contents' }}>
            {textLines.map((line, lineIdx) => {
              const trimmed = line.trim()
              if (!trimmed) {
                return <div key={lineIdx} style={{ height: '0.5rem' }} />
              }

              // Headings
              if (trimmed.startsWith('# ')) {
                return <h1 key={lineIdx}>{trimmed.slice(2)}</h1>
              }
              if (trimmed.startsWith('## ')) {
                return <h2 key={lineIdx}>{trimmed.slice(3)}</h2>
              }
              if (trimmed.startsWith('### ')) {
                return <h3 key={lineIdx}>{trimmed.slice(4)}</h3>
              }

              // List Items
              if (trimmed.startsWith('- ') || trimmed.startsWith('* ')) {
                const inner = formatInline(escapeHTML(trimmed.slice(2)))
                return (
                  <ul key={lineIdx}>
                    <li dangerouslySetInnerHTML={{ __html: inner }} />
                  </ul>
                )
              }
              if (/^\d+\.\s/.test(trimmed)) {
                const match = trimmed.match(/^(\d+)\.\s(.*)/)
                if (match) {
                  const inner = formatInline(escapeHTML(match[2]))
                  return (
                    <ol key={lineIdx} start={parseInt(match[1])}>
                      <li dangerouslySetInnerHTML={{ __html: inner }} />
                    </ol>
                  )
                }
              }

              // Standard Paragraph
              const innerHtml = formatInline(escapeHTML(line))
              return <p key={lineIdx} dangerouslySetInnerHTML={{ __html: innerHtml }} />
            })}
          </div>
        )
      })}
    </div>
  )
}

export const ChatArea: React.FC<ChatAreaProps> = ({
  activeConvId,
  activeConvTitle,
  messages,
  settings,
  isGenerating,
  streamStatus,
  onSendMessage,
  onStopGenerating,
  onOpenMobileSidebar,
  onOpenKnowledge,
}) => {
  const [input, setInput] = useState('')
  const scrollerRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Scroll to bottom helper
  const scrollToBottom = () => {
    if (scrollerRef.current) {
      scrollerRef.current.scrollTop = scrollerRef.current.scrollHeight
    }
  }

  // Auto-scroll when messages update or generating status changes
  useEffect(() => {
    scrollToBottom()
  }, [messages, isGenerating])

  // Handle textarea expanding height dynamically
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = '24px'
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight - 4, 180)}px`
    }
  }, [input])

  const handleSend = () => {
    if (!input.trim() || isGenerating) return
    onSendMessage(input)
    setInput('')
    if (textareaRef.current) {
      textareaRef.current.focus()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const ragActive = settings?.rag_enabled === 'true'

  return (
    <div className="main-workspace">
      {/* Top Header */}
      <header className="chat-header">
        <div className="header-left">
          <button className="menu-toggle" onClick={onOpenMobileSidebar} title="Open Menu">
            <Menu size={20} />
          </button>
          <span className="active-chat-title">
            {activeConvId ? activeConvTitle : 'Studio Workspace'}
          </span>
        </div>
        <div className="header-right">
          {/* RAG Status Pill */}
          <div className={`rag-pill ${ragActive ? 'active' : 'inactive'}`}>
            <div 
              style={{ 
                width: '6px', 
                height: '6px', 
                borderRadius: '50%', 
                background: ragActive ? 'var(--accent-rag)' : 'var(--text-muted)' 
              }} 
            />
            <span>{ragActive ? 'RAG Mode Active' : 'RAG Disabled'}</span>
          </div>

          {/* Knowledge base navigation trigger */}
          <button className="knowledge-nav-btn" onClick={onOpenKnowledge}>
            <BookOpen size={14} />
            Knowledge Base
          </button>
        </div>
      </header>

      {/* Messages Workspace */}
      <div className="messages-scroller" ref={scrollerRef}>
        {!activeConvId ? (
          /* Welcome Screen */
          <div className="welcome-screen">
            <div className="welcome-logo">
              <Sparkles size={40} className="text-white" />
            </div>
            <h2 className="welcome-title">Welcome to Local RAG Studio</h2>
            <p className="welcome-subtitle">
              A private, state-of-the-art playground. Run high-quality chats, index your internal documents, and vectorize intelligence completely offline using local models.
            </p>

            <div className="welcome-cards">
              <div className="welcome-card" onClick={() => onSendMessage("What is RAG (Retrieval-Augmented Generation)?")}>
                <div className="welcome-card-icon">
                  <Sparkles size={18} />
                </div>
                <h3 className="welcome-card-title">Explore Core RAG</h3>
                <p className="welcome-card-desc">Ask what vector embeddings are and how they enrich prompt contexts.</p>
              </div>

              <div className="welcome-card" onClick={onOpenKnowledge}>
                <div className="welcome-card-icon" style={{ color: 'var(--accent-rag)' }}>
                  <BookOpen size={18} />
                </div>
                <h3 className="welcome-card-title">Ingest Documents</h3>
                <p className="welcome-card-desc">Open the Knowledge Base to vectorize texts and index private knowledge.</p>
              </div>
            </div>
          </div>
        ) : (
          /* Messages Stream */
          messages.map((m) => {
            const isUser = m.role === 'user'
            return (
              <div key={m.id} className={`message-row ${isUser ? 'user' : 'assistant'}`}>
                <div className={`message-avatar ${isUser ? 'user' : 'assistant'}`}>
                  {isUser ? 'U' : 'AI'}
                </div>
                <div className="message-content-wrapper">
                  <div className="message-meta">
                    <span>{isUser ? 'User' : 'Assistant'}</span>
                    <span>•</span>
                    <span>{new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                  </div>
                  <div className="message-bubble">
                    <MarkdownRenderer text={m.content} />
                  </div>
                </div>
              </div>
            )
          })
        )}

        {/* Streaming / Typing indicator */}
        {isGenerating && (
          <div className="message-row assistant">
            <div className="message-avatar assistant">AI</div>
            <div className="message-content-wrapper">
              <div className="message-meta">
                <span>Assistant</span>
                <span>•</span>
                <span>{streamStatus === 'searching' ? 'Searching knowledge base...' : streamStatus === 'generating' ? 'Generating...' : 'Thinking...'}</span>
              </div>
              <div className="message-bubble" style={{ padding: '0.75rem 1rem' }}>
                <div className="typing-indicator">
                  <div className="typing-dot" />
                  <div className="typing-dot" />
                  <div className="typing-dot" />
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Input panel */}
      <footer className="chat-input-panel">
        <div className="input-container">
          <textarea
            ref={textareaRef}
            className="chat-textarea"
            placeholder={
              !activeConvId 
                ? "Create or select a thread from the sidebar to start writing..." 
                : isGenerating 
                  ? "Assistant is answering your prompt..." 
                  : "Write your prompt, press Enter to submit..."
            }
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={!activeConvId || isGenerating}
          />
          {isGenerating ? (
            <button
              className="send-btn"
              onClick={onStopGenerating}
              title="Stop generating"
              style={{ background: '#ef4444' }}
            >
              <Square size={16} fill="currentColor" />
            </button>
          ) : (
            <button
              className="send-btn"
              onClick={handleSend}
              disabled={!input.trim() || !activeConvId}
              title="Send Message"
            >
              <Send size={16} />
            </button>
          )}
        </div>
      </footer>
    </div>
  )
}
