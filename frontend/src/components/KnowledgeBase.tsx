import React, { useState, useEffect, useRef } from 'react'
import { X, BookOpen, Upload, Trash2, FileText, Sparkles, AlertCircle, Plus } from 'lucide-react'
import { DocumentInfo } from '../types'

interface KnowledgeBaseProps {
  onClose: () => void;
}

export const KnowledgeBase: React.FC<KnowledgeBaseProps> = ({ onClose }) => {
  const [documents, setDocuments] = useState<DocumentInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [isVectorizing, setIsVectorizing] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  // Paste / File review editor form state
  const [showEditor, setShowEditor] = useState(false)
  const [docTitle, setDocTitle] = useState('')
  const [docContent, setDocContent] = useState('')

  const fileInputRef = useRef<HTMLInputElement>(null)

  // Fetch all documents on mount
  useEffect(() => {
    fetchDocuments()
  }, [])

  const fetchDocuments = async () => {
    setLoading(true)
    setErrorMessage('')
    try {
      const res = await fetch('/api/knowledge')
      if (!res.ok) throw new Error('Failed to retrieve index')
      const data = await res.json()
      setDocuments(data || [])
    } catch (err: any) {
      setErrorMessage(err.message || 'ChromaDB is currently unreachable.')
    } finally {
      setLoading(false)
    }
  }

  // Handle local file read
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const text = event.target?.result as string
      // Auto-populate paste editor
      setDocTitle(file.name.replace(/\.[^/.]+$/, "")) // strip extension
      setDocContent(text)
      setShowEditor(true)
      setErrorMessage('')
      setSuccessMessage('')
    }
    reader.readAsText(file)
  }

  // Handle vectorize submit
  const handleVectorize = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!docTitle.trim() || !docContent.trim() || isVectorizing) return

    setIsVectorizing(true)
    setErrorMessage('')
    setSuccessMessage('')

    try {
      const res = await fetch('/api/knowledge', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: docTitle, content: docContent }),
      })

      if (!res.ok) {
        const errText = await res.text()
        throw new Error(errText || 'Vector indexing failed')
      }

      const data = await res.json()
      setSuccessMessage(`Document successfully vectorized into ${data.chunks} semantic vectors!`)
      setDocTitle('')
      setDocContent('')
      setShowEditor(false)
      fetchDocuments()
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to submit document to vector pipeline')
    } finally {
      setIsVectorizing(false)
    }
  }

  // Handle document deletion
  const handleDeleteDoc = async (id: string) => {
    if (!window.confirm("Are you sure you want to delete this document from the vector collection? This removes all associated semantic vectors.")) return
    setErrorMessage('')
    setSuccessMessage('')
    try {
      const res = await fetch(`/api/knowledge/${id}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Deletion failed')
      setSuccessMessage('Document removed from vector store.')
      fetchDocuments()
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to delete vectors.')
    }
  }

  const handleDeleteAll = async () => {
    if (!window.confirm(`WARNING: This will permanently delete ALL ${documents.length} documents and their vectors from the knowledge base. This cannot be undone. Are you sure?`)) return
    setErrorMessage('')
    setSuccessMessage('')
    setLoading(true)
    try {
      const res = await fetch('/api/knowledge', {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Failed to clear vector library')
      const data = await res.json()
      setSuccessMessage(`Cleared ${data.deleted || documents.length} documents from vector store.`)
      setDocuments([])
    } catch (err: any) {
      setErrorMessage(err.message || 'Failed to clear vector library.')
    } finally {
      setLoading(false)
    }
  }

  const triggerFileSelect = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click()
    }
  }

  return (
    <div className="knowledge-panel">
      {/* Header */}
      <header className="kpanel-header">
        <div className="kpanel-title-area">
          <BookOpen size={18} className="text-indigo-400" />
          <span className="kpanel-title">Knowledge Management Studio</span>
        </div>
        <button className="close-kpanel-btn" onClick={onClose} title="Return to Workspace">
          <X size={16} />
        </button>
      </header>

      {/* Main Workspace Scroll area */}
      <div className="kpanel-content">
        
        {/* Alerts for feedback */}
        {errorMessage && (
          <div style={{ background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.3)', padding: '0.85rem 1rem', borderRadius: '12px', color: '#f87171', display: 'flex', alignItems: 'center', gap: '0.6rem', fontSize: '0.82rem' }}>
            <AlertCircle size={16} style={{ flexShrink: 0 }} />
            <span>{errorMessage}</span>
          </div>
        )}

        {successMessage && (
          <div style={{ background: 'rgba(16,185,129,0.1)', border: '1px solid rgba(16,185,129,0.3)', padding: '0.85rem 1rem', borderRadius: '12px', color: '#34d399', display: 'flex', alignItems: 'center', gap: '0.6rem', fontSize: '0.82rem' }}>
            <Sparkles size={16} style={{ flexShrink: 0 }} />
            <span>{successMessage}</span>
          </div>
        )}

        <div className="kpanel-grid">
          
          {/* Column 1: Document Ingest */}
          <div className="upload-card">
            <h3 className="section-title">Ingest intelligence</h3>
            <p className="section-subtitle">Vectorize texts or local files to feed the local RAG pipeline context.</p>

            {!showEditor ? (
              /* Initial Drag zone screen */
              <>
                <div className="upload-dropzone" onClick={triggerFileSelect}>
                  <Upload size={32} className="upload-dropzone-icon" />
                  <p className="upload-text">Upload Local Files</p>
                  <p className="upload-subtext">Click to choose a .txt, .md, or text file</p>
                  <input
                    type="file"
                    ref={fileInputRef}
                    onChange={handleFileSelect}
                    style={{ display: 'none' }}
                    accept=".txt,.md,.json,.js,.ts,.go,.html,.css,.yaml,.yml"
                  />
                </div>

                <button className="paste-trigger" onClick={() => setShowEditor(true)}>
                  <Plus size={12} style={{ marginRight: '0.4rem' }} />
                  Paste Raw Text Instead
                </button>
              </>
            ) : (
              /* Review Editor screen */
              <form onSubmit={handleVectorize} className="paste-editor">
                <input
                  type="text"
                  className="paste-input"
                  placeholder="Document Title (e.g. Server Guide)"
                  value={docTitle}
                  onChange={(e) => setDocTitle(e.target.value)}
                  disabled={isVectorizing}
                  required
                />
                <textarea
                  className="paste-textarea"
                  placeholder="Paste your custom document body content here... Ollama will chunk and vectorize it semantic-by-semantic."
                  value={docContent}
                  onChange={(e) => setDocContent(e.target.value)}
                  disabled={isVectorizing}
                  required
                />
                <div className="paste-actions">
                  <button
                    type="button"
                    className="btn-secondary"
                    onClick={() => {
                      setShowEditor(false)
                      setDocTitle('')
                      setDocContent('')
                    }}
                    disabled={isVectorizing}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn-primary"
                    disabled={!docTitle.trim() || !docContent.trim() || isVectorizing}
                  >
                    {isVectorizing ? 'Vectorizing intelligence...' : 'Vectorize Document'}
                  </button>
                </div>
              </form>
            )}
          </div>

          {/* Column 2: Vector Library */}
          <div className="documents-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
              <h3 className="section-title" style={{ margin: 0 }}>Vector Library</h3>
              {documents.length > 0 && (
                <button
                  onClick={handleDeleteAll}
                  disabled={loading}
                  title="Delete all documents and vectors"
                  style={{
                    display: 'flex', alignItems: 'center', gap: '0.35rem',
                    background: 'rgba(239,68,68,0.15)', border: '1px solid rgba(239,68,68,0.3)',
                    color: '#f87171', padding: '0.35rem 0.7rem', borderRadius: '8px',
                    fontSize: '0.72rem', cursor: 'pointer', transition: 'all 0.2s',
                  }}
                >
                  <Trash2 size={12} />
                  Clear All
                </button>
              )}
            </div>
            <p className="section-subtitle">ChromaDB index of semantic document structures.</p>

            {loading ? (
              <div className="empty-docs">
                <div className="typing-indicator" style={{ marginBottom: '0.5rem' }}>
                  <div className="typing-dot" />
                  <div className="typing-dot" />
                  <div className="typing-dot" />
                </div>
                <p style={{ fontSize: '0.8rem' }}>Loading index...</p>
              </div>
            ) : documents.length === 0 ? (
              <div className="empty-docs">
                <FileText size={32} className="empty-docs-icon" />
                <p className="empty-docs-text">
                  Vector index is empty. Ingest text or documents to enable semantic RAG lookups in your threads!
                </p>
              </div>
            ) : (
              <div className="docs-list">
                {documents.map((doc) => (
                  <div key={doc.id} className="doc-item">
                    <div className="doc-info">
                      <div className="doc-icon">
                        <FileText size={16} />
                      </div>
                      <div className="doc-metadata">
                        <span className="doc-title" title={doc.title}>{doc.title}</span>
                        <div className="doc-stats">
                          <span>{doc.chunks} vectors</span>
                          <span>•</span>
                          <span>{new Date(doc.created_at).toLocaleDateString()}</span>
                        </div>
                      </div>
                    </div>
                    <button
                      className="doc-delete-btn"
                      onClick={() => handleDeleteDoc(doc.id)}
                      title="Delete document and erase vectors"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

        </div>
      </div>
    </div>
  )
}
