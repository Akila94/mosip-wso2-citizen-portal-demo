import React from 'react';
import { Placeholder } from '../common/Placeholder';

/**
 * Frame 12 — modal, not a redirect. Focus intent: trap focus while open,
 * mark the page behind aria-hidden, and disable Escape (no safe cancel),
 * per the wireframe's a11y note.
 */
export function SessionExpiredModal({ onStartNewSession, onBackToPortal }: { onStartNewSession: () => void; onBackToPortal: () => void }) {
  const dialogRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(13,13,13,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 'var(--space-6)', zIndex: 100 }}
      role="presentation"
    >
      <div
        ref={dialogRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="session-expired-title"
        tabIndex={-1}
        style={{ width: 460, maxWidth: '100%', background: 'var(--surface-card)', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-lg)', padding: 'var(--space-6)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
          <Placeholder label="IS" width={30} height={30} />
          <h2 id="session-expired-title" style={{ margin: 0, font: '600 16px var(--font-sans)' }}>Your session has expired</h2>
        </div>
        <p style={{ margin: 0, font: '400 12.5px/1.6 var(--font-sans)', color: 'var(--text-secondary)' }}>
          You were inactive for 15 minutes, so we signed you out to protect your details. Your answers to steps 1 and 2 are saved as a draft.
        </p>
        <div style={{ border: '1px solid var(--border-default)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', background: 'var(--surface-sunken)', font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>
          draft ref DL-DRAFT-004871 · saved 14:02
        </div>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <button type="button" onClick={onStartNewSession} style={{ flex: 1, padding: 'var(--space-3)', background: 'var(--gray-950)', color: 'var(--gray-0)', border: 'none', borderRadius: 'var(--radius-sm)', font: '600 13.5px var(--font-sans)', cursor: 'pointer' }}>
            Start new session
          </button>
          <button type="button" onClick={onBackToPortal} style={{ padding: 'var(--space-3) var(--space-4)', background: 'transparent', color: 'var(--text-primary)', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', font: '600 13.5px var(--font-sans)', cursor: 'pointer' }}>
            Back to portal
          </button>
        </div>
      </div>
    </div>
  );
}
