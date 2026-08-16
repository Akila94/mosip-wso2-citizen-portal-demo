import React from 'react';

export function SessionInspectorToggle({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      style={{ marginLeft: 'auto', background: 'none', border: '1.5px dotted var(--border-strong)', borderRadius: 'var(--radius-sm)', padding: '6px 10px', font: '500 11px var(--font-mono)', color: 'var(--text-secondary)', cursor: 'pointer' }}
    >
      ⌥ Session inspector {open ? '▾' : '▸'}
    </button>
  );
}
