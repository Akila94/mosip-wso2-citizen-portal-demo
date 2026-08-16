import React, { useEffect, useRef, useState } from 'react';
import { Avatar } from '../../design-system/components/core/Avatar';
import { Card } from '../../design-system/components/core/Card';
import { useAuth } from '../../context/AuthContext';
import type { Screen } from '../../App';

const MENU_ITEMS: { label: string; screen: Screen }[] = [
  { label: 'My Timeline', screen: 'timeline' },
  { label: 'My Documents', screen: 'documents' },
  { label: 'Profile & Consents', screen: 'profile' },
];

export interface AccountMenuProps {
  onNavigate: (screen: Screen) => void;
}

export function AccountMenu({ onNavigate }: AccountMenuProps) {
  const { user, signOut } = useAuth();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  if (!user) return null;

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 9,
          padding: '6px 12px 6px 6px',
          border: '1.5px solid var(--border-strong)',
          borderRadius: 'var(--radius-pill)',
          background: 'var(--surface-card)',
          cursor: 'pointer',
        }}
      >
        <Avatar name={user.displayName} size="sm" />
        <span style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-primary)' }}>{user.displayName}</span>
        <span aria-hidden="true" style={{ fontSize: 11, color: 'var(--text-secondary)' }}>▾</span>
      </button>
      {open && (
        <Card role="menu" padded={false} style={{ position: 'absolute', right: 0, top: 'calc(100% + 8px)', width: 220, zIndex: 20, overflow: 'hidden' }}>
          {MENU_ITEMS.map((item) => (
            <button
              key={item.label}
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false);
                onNavigate(item.screen);
              }}
              style={{ display: 'block', width: '100%', textAlign: 'left', padding: '13px 14px', border: 'none', borderBottom: '1px solid var(--border-subtle)', background: 'var(--surface-card)', font: '600 13px var(--font-sans)', color: 'var(--text-primary)', cursor: 'pointer' }}
            >
              {item.label}
            </button>
          ))}
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setOpen(false);
              signOut();
            }}
            style={{ display: 'block', width: '100%', textAlign: 'left', padding: '13px 14px', border: 'none', background: 'var(--surface-card)', font: '600 13px var(--font-sans)', color: 'var(--danger)', cursor: 'pointer' }}
          >
            Sign out
          </button>
        </Card>
      )}
    </div>
  );
}
