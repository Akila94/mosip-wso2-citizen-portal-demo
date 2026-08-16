import React from 'react';
import type { AssuranceRequirement } from '../../services/types';

const LEVELS: { id: AssuranceRequirement; label: string }[] = [
  { id: 'low', label: 'Low' },
  { id: 'substantial', label: 'Substantial' },
  { id: 'high', label: 'High' },
];

/** Deviation: no segmented-control primitive in the design system — built
 * from DS tokens, consistent with the same pattern already used for the
 * language switcher and payment-method picker elsewhere in this app. */
export function AssuranceLevelPicker({ value, onChange }: { value: AssuranceRequirement; onChange: (v: AssuranceRequirement) => void }) {
  return (
    <div role="radiogroup" aria-label="Assurance level required" style={{ display: 'flex', gap: 'var(--space-2)' }}>
      {LEVELS.map((l) => {
        const active = l.id === value;
        return (
          <button
            key={l.id}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(l.id)}
            style={{ flex: 1, textAlign: 'center', padding: 11, border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', font: '600 12px var(--font-sans)', background: active ? 'var(--gray-950)' : 'var(--surface-card)', color: active ? 'var(--gray-0)' : 'var(--text-primary)', cursor: 'pointer' }}
          >
            {l.label}
          </button>
        );
      })}
    </div>
  );
}
