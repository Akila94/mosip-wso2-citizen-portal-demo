import React from 'react';
import { Checkbox } from '../../design-system/components/forms/Checkbox';
import type { ScopeOption } from '../../services/types';

export function ScopeChecklist({ scopes, selectedIds, onToggle }: { scopes: ScopeOption[]; selectedIds: string[]; onToggle: (id: string) => void }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
      {scopes.map((s) => (
        <div key={s.id} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)' }}>
          <Checkbox checked={selectedIds.includes(s.id)} onChange={() => onToggle(s.id)} />
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ font: '500 12px var(--font-mono)' }}>{s.label}</span>
            <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>{s.description}</span>
          </div>
        </div>
      ))}
      <span style={{ borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
        every scope requested here becomes a line on the citizen's consent screen and in Profile &amp; Consents
      </span>
    </div>
  );
}
