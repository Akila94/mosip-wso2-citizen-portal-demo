import React, { useState } from 'react';
import { MessageCircle, X, Send } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { useAssistant } from '../../hooks/useAssistant';
import { Button } from '../../design-system/components/core/Button';

export interface AiAssistantWidgetProps {
  onStartRenewal: () => void;
}

/**
 * Docked bottom-right on every screen, per the wireframe. Deviation: the
 * design system has no chat/message-bubble component, so the panel and its
 * bubbles are built from DS tokens (surface/border/radius/spacing) rather
 * than a native DS primitive.
 */
export function AiAssistantWidget({ onStartRenewal }: AiAssistantWidgetProps) {
  const { isAuthenticated } = useAuth();
  const { open, setOpen, messages, sending, error, send } = useAssistant(isAuthenticated);
  const [draft, setDraft] = useState('');

  function handleSend(text?: string) {
    const q = text ?? draft;
    if (!q.trim()) return;
    send(q);
    setDraft('');
  }

  return (
    <div style={{ position: 'fixed', right: 'var(--space-5)', bottom: 'var(--space-5)', zIndex: 60, display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 'var(--space-3)' }}>
      {open && (
        <div role="dialog" aria-label="Ask MaroliaGov" style={{ width: 360, maxHeight: 460, display: 'flex', flexDirection: 'column', background: 'var(--surface-card)', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-lg)', overflow: 'hidden' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 'var(--space-3) var(--space-4)', borderBottom: '1px solid var(--border-default)' }}>
            <span style={{ font: '600 13px var(--font-sans)' }}>Ask MaroliaGov</span>
            <button type="button" onClick={() => setOpen(false)} aria-label="Close" style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', display: 'flex' }}>
              <X size={16} />
            </button>
          </div>
          <div style={{ flex: 1, overflowY: 'auto', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
            {messages.length === 0 && (
              <p style={{ margin: 0, font: '400 12.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
                Ask a question about any government service{!isAuthenticated ? ' — sign in for answers about your own records.' : '.'}
              </p>
            )}
            {messages.map((m) => (
              <div key={m.id} style={{ alignSelf: m.role === 'user' ? 'flex-end' : 'flex-start', maxWidth: '88%', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <div style={{ border: '1px solid var(--border-default)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', background: m.role === 'user' ? 'var(--surface-sunken)' : 'var(--surface-card)', font: '400 12.5px/1.5 var(--font-sans)' }}>
                  {m.text}
                </div>
                {m.actions && (
                  <div style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
                    {m.actions.map((a) => (
                      <button
                        key={a.label}
                        type="button"
                        onClick={() => (a.kind === 'start-renewal' ? onStartRenewal() : handleSend(a.label))}
                        style={{ padding: '8px 12px', borderRadius: 'var(--radius-pill)', border: a.kind === 'start-renewal' ? 'none' : '1.5px solid var(--border-strong)', background: a.kind === 'start-renewal' ? 'var(--gray-950)' : 'transparent', color: a.kind === 'start-renewal' ? 'var(--gray-0)' : 'var(--text-primary)', font: '600 11.5px var(--font-sans)', cursor: 'pointer' }}
                      >
                        {a.label}
                      </button>
                    ))}
                  </div>
                )}
                {m.sourceNote && <span style={{ font: '400 10.5px var(--font-mono)', color: 'var(--text-muted)' }}>{m.sourceNote}</span>}
              </div>
            ))}
            {sending && <span style={{ font: '400 12px var(--font-sans)', color: 'var(--text-secondary)' }}>Thinking…</span>}
            {error && <span style={{ font: '400 12px var(--font-sans)', color: 'var(--danger)' }}>{error}</span>}
          </div>
          <div style={{ display: 'flex', gap: 'var(--space-2)', padding: 'var(--space-3)', borderTop: '1px solid var(--border-default)' }}>
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSend()}
              placeholder="Ask a question"
              aria-label="Ask a question"
              style={{ flex: 1, border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: '10px 12px', font: '400 13px var(--font-sans)', background: 'var(--surface-card)', color: 'var(--text-primary)' }}
            />
            <Button size="sm" onClick={() => handleSend()} aria-label="Send">
              <Send size={14} />
            </Button>
          </div>
        </div>
      )}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={open ? 'Close assistant' : 'Open assistant'}
        style={{ width: 56, height: 56, borderRadius: '50%', border: 'none', background: 'var(--gray-950)', color: 'var(--gray-0)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', boxShadow: 'var(--shadow-lg)' }}
      >
        {open ? <X size={22} /> : <MessageCircle size={22} />}
      </button>
    </div>
  );
}
