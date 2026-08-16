import React from 'react';
import { Placeholder } from '../common/Placeholder';

/** External national eID provider stub — a different hostname/chrome than
 * both the portal and WSO2 Identity Server, on purpose: it makes the
 * federation hand-off visually obvious. */
export function ExternalIdpShell({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ minHeight: '100vh', background: 'var(--surface-page)', padding: 'var(--space-6)', display: 'flex', justifyContent: 'center' }}>
      <div style={{ width: '100%', maxWidth: 640, border: '2px dashed var(--border-default)', borderRadius: 'var(--radius-lg)', padding: 'var(--space-3)', background: 'var(--gray-100, var(--surface-sunken))' }}>
        <div style={{ padding: '0 var(--space-2) var(--space-2)', font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
          External trust boundary — national eID provider
        </div>
        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
          <header style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)', padding: 'var(--space-4) var(--space-6)', borderBottom: '1px solid var(--border-default)' }}>
            <Placeholder label="eID" width={32} height={32} />
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              <span style={{ font: '600 13.5px var(--font-sans)' }}>Marolia National Digital ID</span>
              <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>eid.marolia.gov · external provider</span>
            </div>
          </header>
          {children}
        </div>
      </div>
    </div>
  );
}
