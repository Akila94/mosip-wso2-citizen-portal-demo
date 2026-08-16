import React from 'react';
import { Placeholder } from '../common/Placeholder';

export interface IdentityShellProps {
  serviceName?: string;
  children: React.ReactNode;
}

/**
 * WSO2 Identity Server's own chrome — a different product than the portal,
 * so it deliberately does not reuse AppHeader/AppFooter. The dashed outer
 * border is the wireframe's "trust boundary" convention: everything inside
 * is served by the identity server, not the portal.
 */
export function IdentityShell({ serviceName, children }: IdentityShellProps) {
  return (
    <div style={{ minHeight: '100vh', background: 'var(--surface-page)', padding: 'var(--space-6)', display: 'flex', justifyContent: 'center' }}>
      <div style={{ width: '100%', maxWidth: 1100, border: '2px dashed var(--border-strong)', borderRadius: 'var(--radius-lg)', padding: 'var(--space-3)', background: 'var(--surface-sunken)' }}>
        <div style={{ padding: '0 var(--space-2) var(--space-2)', font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
          Trust boundary — WSO2 Identity Server · identity.marolia.gov
        </div>
        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
          <header style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)', padding: 'var(--space-4) var(--space-6)', borderBottom: '1px solid var(--border-default)' }}>
            <Placeholder label="IS" width={34} height={34} />
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              <span style={{ font: '600 14px var(--font-sans)' }}>Marolia Digital Identity</span>
              {serviceName && <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>signing in to {serviceName}</span>}
            </div>
          </header>
          {children}
        </div>
      </div>
    </div>
  );
}
