import React from 'react';
import { AccessibilityToolbar } from './AccessibilityToolbar';

export function AppFooter({ compact = false }: { compact?: boolean }) {
  return (
    <footer style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) var(--space-8)', borderTop: '1px solid var(--border-default)', background: 'var(--surface-sunken)', flexWrap: 'wrap' }}>
      <span style={{ font: '600 11px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Accessibility</span>
      <AccessibilityToolbar variant={compact ? 'compact' : 'full'} />
    </footer>
  );
}
