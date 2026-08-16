import React from 'react';

export interface RadioOptionProps {
  name: string;
  label: string;
  hint?: string;
  checked: boolean;
  onChange: () => void;
}

/**
 * Deviation: the design system has no radio-button primitive (Checkbox and
 * Switch only). Built from DS tokens as an accessible native radio input
 * styled to match card rows, rather than inventing a visual language.
 */
export function RadioOption({ name, label, hint, checked, onChange }: RadioOptionProps) {
  return (
    <label style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)', border: `1.5px solid ${checked ? 'var(--accent)' : 'var(--border-strong)'}`, borderRadius: 'var(--radius-md)', padding: 'var(--space-3) var(--space-4)', cursor: 'pointer' }}>
      <input type="radio" name={name} checked={checked} onChange={onChange} style={{ width: 16, height: 16, accentColor: 'var(--accent)', flexShrink: 0 }} />
      <div style={{ display: 'flex', flexDirection: 'column' }}>
        <span style={{ font: '600 13px var(--font-sans)' }}>{label}</span>
        {hint && <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{hint}</span>}
      </div>
    </label>
  );
}
