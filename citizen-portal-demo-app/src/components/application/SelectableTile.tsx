import React from 'react';

export interface SelectableTileProps {
  label: string;
  description: string;
  selected: boolean;
  onSelect: () => void;
  type?: 'radio' | 'checkbox';
  name?: string;
  disabled?: boolean;
  disabledReason?: string;
}

/** Deviation: the design system has no selectable-card primitive (Checkbox
 * and radio inputs are plain rows). Built from native radio/checkbox inputs
 * styled as bordered tiles with DS tokens, matching the wireframe's card
 * layout for application type and licence class pickers. */
export function SelectableTile({ label, description, selected, onSelect, type = 'checkbox', name, disabled, disabledReason }: SelectableTileProps) {
  return (
    <label
      style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--space-3)',
        border: `1.5px solid ${selected ? 'var(--accent)' : 'var(--border-strong)'}`,
        borderRadius: 'var(--radius-md)',
        padding: 'var(--space-3) var(--space-4)',
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.55 : 1,
        background: 'var(--surface-card)',
      }}
      title={disabled ? disabledReason : undefined}
    >
      <input type={type} name={name} checked={selected} disabled={disabled} onChange={onSelect} style={{ width: 17, height: 17, accentColor: 'var(--accent)', flexShrink: 0 }} />
      <div style={{ display: 'flex', flexDirection: 'column' }}>
        <span style={{ font: '600 13px var(--font-sans)' }}>{label}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{description}</span>
      </div>
    </label>
  );
}
