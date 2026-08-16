import React from 'react';
import type { ReviewSection, VerifiedIdentitySummary } from '../../services/types';

export function ApplicationReview({ sections, identity }: { sections: ReviewSection[]; identity: VerifiedIdentitySummary }) {
  return (
    <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
      {sections.map((s) => (
        <div key={s.step} style={{ display: 'grid', gridTemplateColumns: '150px 1fr 70px', gap: 'var(--space-4)', padding: 'var(--space-4)', borderBottom: '1px dashed var(--border-default)' }}>
          <span style={{ font: '500 11px var(--font-mono)', letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>{s.step}</span>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {s.lines.map((line, i) => (
              <span key={i} style={{ font: '400 12.5px/1.5 var(--font-sans)' }}>{line}</span>
            ))}
          </div>
          <button type="button" style={{ background: 'none', border: 'none', padding: 0, font: '600 12px var(--font-sans)', textDecoration: 'underline', textAlign: 'right', cursor: 'pointer' }}>Edit</button>
        </div>
      ))}
      <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr 70px', gap: 'var(--space-4)', padding: 'var(--space-4)', background: 'var(--surface-sunken)' }}>
        <span style={{ font: '500 11px var(--font-mono)', letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Identity</span>
        <span style={{ font: '400 12.5px/1.5 var(--font-sans)' }}>
          {identity.name} · {identity.attributes.map((a) => a.value).join(' · ')} — <strong>verified, National Digital ID</strong>
        </span>
        <span style={{ font: '500 11.5px var(--font-mono)', color: 'var(--text-muted)', textAlign: 'right' }}>🔒</span>
      </div>
    </div>
  );
}
