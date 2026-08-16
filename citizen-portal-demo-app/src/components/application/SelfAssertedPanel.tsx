import React from 'react';
import type { SelfAssertedAttribute } from '../../services/types';

export function SelfAssertedPanel({ attributes, onEdit }: { attributes: SelfAssertedAttribute[]; onEdit?: (id: string) => void }) {
  return (
    <div style={{ border: '1px solid var(--border-default)', background: 'var(--surface-card)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
      <span style={{ font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>You told us</span>
      {attributes.map((a) => (
        <div key={a.id} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-2)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)' }}>
          <span style={{ font: '400 11.5px var(--font-sans)', color: 'var(--text-secondary)' }}>{a.label}</span>
          <span style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
            <span style={{ font: '600 11.5px var(--font-sans)' }}>{a.value}</span>
            <button type="button" onClick={() => onEdit?.(a.id)} style={{ background: 'none', border: 'none', padding: 0, font: '500 10.5px var(--font-sans)', textDecoration: 'underline', color: 'var(--text-link)', cursor: 'pointer' }}>edit</button>
          </span>
        </div>
      ))}
      <span style={{ borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
        self-asserted · editable · not verified by any authority
      </span>
    </div>
  );
}
