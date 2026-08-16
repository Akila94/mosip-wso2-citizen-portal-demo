import React from 'react';
import type { AdminNavItem } from '../../services/types';

export function AdminSidebarNav({ items, activeId }: { items: AdminNavItem[]; activeId: string }) {
  return (
    <nav aria-label="Console sections" style={{ width: 230, flexShrink: 0, borderRight: '1px solid var(--border-default)', background: 'var(--surface-sunken)', padding: 'var(--space-5) var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
      <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>Console</span>
      {items.map((item) => {
        const active = item.id === activeId;
        return (
          <div
            key={item.id}
            aria-current={active ? 'page' : undefined}
            style={{
              padding: '11px 12px',
              border: '1.5px solid var(--border-strong)',
              borderRadius: 'var(--radius-sm)',
              font: '600 12.5px var(--font-sans)',
              background: active ? 'var(--gray-950)' : 'var(--surface-card)',
              color: active ? 'var(--gray-0)' : 'var(--text-primary)',
            }}
          >
            {item.label}
          </div>
        );
      })}
    </nav>
  );
}
