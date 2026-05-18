import React from 'react'
import { Plus, MessageSquare, Trash2, Settings, Compass, Sparkles } from 'lucide-react'
import { Conversation } from '../types'

interface SidebarProps {
  conversations: Conversation[];
  activeConvId: string | null;
  onSelectConv: (id: string) => void;
  onNewConv: () => void;
  onDeleteConv: (id: string, e: React.MouseEvent) => void;
  onOpenSettings: () => void;
  isMobileOpen: boolean;
  onCloseMobile: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  conversations,
  activeConvId,
  onSelectConv,
  onNewConv,
  onDeleteConv,
  onOpenSettings,
  isMobileOpen,
  onCloseMobile,
}) => {
  return (
    <>
      {/* Mobile Drawer Overlay */}
      <div 
        className={`sidebar-overlay ${isMobileOpen ? 'mobile-open' : ''}`}
        onClick={onCloseMobile}
      />

      <aside className={`sidebar ${isMobileOpen ? 'mobile-open' : ''}`}>
        {/* Header Title */}
        <div className="sidebar-header">
          <div className="logo-icon">
            <Sparkles size={20} className="text-white animate-pulse" />
          </div>
          <div>
            <h1 className="logo-text">AI RAG Studio</h1>
            <p style={{ fontSize: '0.68rem', color: 'var(--text-muted)', fontWeight: 500 }}>OLLAMA & CHROMADB</p>
          </div>
        </div>

        {/* New Chat trigger */}
        <button className="new-chat-btn" onClick={() => {
          onNewConv();
          onCloseMobile();
        }}>
          <Plus size={18} />
          New Chat Thread
        </button>

        {/* Threads List */}
        <div className="threads-container">
          {conversations.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '2rem 1rem', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
              <Compass size={24} style={{ marginBottom: '0.5rem', opacity: 0.3 }} />
              <p>No chat history yet.</p>
              <p style={{ fontSize: '0.72rem', marginTop: '0.25rem' }}>Create a thread to begin!</p>
            </div>
          ) : (
            conversations.map((c) => (
              <div
                key={c.id}
                className={`thread-item ${activeConvId === c.id ? 'active' : ''}`}
                onClick={() => {
                  onSelectConv(c.id);
                  onCloseMobile();
                }}
              >
                <div className="thread-details">
                  <MessageSquare size={16} style={{ flexShrink: 0 }} />
                  <span className="thread-title">{c.title}</span>
                </div>
                <button
                  className="thread-delete-btn"
                  onClick={(e) => onDeleteConv(c.id, e)}
                  title="Delete Thread"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))
          )}
        </div>

        {/* Footer Actions */}
        <div className="sidebar-footer">
          <div className="settings-trigger" onClick={onOpenSettings}>
            <div className="settings-trigger-left">
              <Settings size={16} className="text-indigo-400" />
              <span>Studio Settings</span>
            </div>
            <span style={{ fontSize: '0.7rem', color: 'var(--text-muted)', background: 'rgba(255,255,255,0.05)', padding: '0.15rem 0.4rem', borderRadius: '4px' }}>v1.0</span>
          </div>
        </div>
      </aside>
    </>
  )
}
