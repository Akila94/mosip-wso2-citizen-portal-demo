import React from 'react';
import { Switch } from '../../design-system/components/forms/Switch';
import type { ConsentAttributeRequest } from '../../services/types';

export function ConsentAttributeRow({ attribute, allowed, onToggle }: { attribute: ConsentAttributeRequest; allowed: boolean; onToggle: (allowed: boolean) => void }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 'var(--space-4)', alignItems: 'center', padding: 'var(--space-3) var(--space-4)', borderTop: '1px dashed var(--border-default)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
          <span style={{ font: '600 13.5px var(--font-sans)' }}>{attribute.label}</span>
          <span style={{ font: '500 10px var(--font-mono)', padding: '3px 5px', border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-sm)' }}>{attribute.required ? 'REQUIRED' : 'OPTIONAL'}</span>
        </div>
        <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>{attribute.value} · source: {attribute.source}</span>
      </div>
      <Switch checked={allowed} onChange={() => onToggle(!allowed)} aria-label={`${allowed ? 'Allow' : 'Deny'} sharing ${attribute.label}`} />
    </div>
  );
}
