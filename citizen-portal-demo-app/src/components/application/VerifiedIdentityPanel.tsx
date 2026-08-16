import React from 'react';
import { Placeholder } from '../common/Placeholder';
import type { VerifiedIdentitySummary } from '../../services/types';

/**
 * Persistent read-only panel present from step 1 onward, per the wireframe's
 * central rule: identity-derived facts are never repeated as editable or
 * greyed-out inputs — they live here, stated as verified and released by
 * consent, not re-collected.
 */
export function VerifiedIdentityPanel({ identity, showSessionNote }: { identity: VerifiedIdentitySummary; showSessionNote?: boolean }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
      <span style={{ font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Verified identity</span>
      <div style={{ border: '1px solid var(--border-default)', background: 'var(--surface-card)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
        <div style={{ display: 'flex', gap: 'var(--space-3)', alignItems: 'center' }}>
          <Placeholder label="photo from ID" width={62} height={74} />
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ font: '600 14px var(--font-sans)' }}>{identity.name}</span>
            <span style={{ font: '500 10px var(--font-mono)', padding: '4px 6px', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-pill)', alignSelf: 'flex-start' }}>🔒 {identity.badgeLabel}</span>
          </div>
        </div>
        {identity.attributes.map((a, i) => (
          <div key={i} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-2)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)' }}>
            <span style={{ font: '400 11.5px var(--font-sans)', color: 'var(--text-secondary)' }}>{a.label}</span>
            <span style={{ font: '600 11.5px var(--font-sans)', textAlign: 'right' }}>{a.value}</span>
          </div>
        ))}
        <div style={{ borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
          read-only · released by the identity service after your consent · not editable here
        </div>
      </div>
      <div style={{ border: '1px dotted var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
        <span style={{ font: '600 12px var(--font-sans)' }}>Something wrong?</span>
        <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-primary)' }}>
          These facts come from your national identity record. Correct them with the registry — the correction flows back here. The application cannot overwrite them.
        </span>
        <a href="#correct-record" style={{ font: '600 12px var(--font-sans)' }}>How to correct my record</a>
      </div>
      {showSessionNote && (
        <div style={{ border: '1px dotted var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-primary)' }}>
          <strong>Session</strong> · signed in 14:02 · expires in 57 min · adaptive step-up will be requested at step 4.
        </div>
      )}
    </div>
  );
}
