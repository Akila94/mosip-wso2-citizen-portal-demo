import React, { useState } from 'react';

const LANGUAGES = [
  { code: 'EN', label: 'English' },
  { code: 'SI', label: 'Sinhala' },
  { code: 'TA', label: 'Tamil' },
];

export interface LanguageSwitcherProps {
  value?: string;
  onChange?: (code: string) => void;
}

/**
 * Deviation: the design system has no segmented-control primitive, so this
 * is hand-built from DS tokens (border/radius/color variables) rather than
 * ad-hoc styling. Content is not translated yet — switching language only
 * changes the active state; wiring real i18n strings is flagged future work.
 */
export function LanguageSwitcher({ value: valueProp, onChange }: LanguageSwitcherProps) {
  const [internal, setInternal] = useState('EN');
  const value = valueProp ?? internal;
  return (
    <div role="radiogroup" aria-label="Select language" style={{ display: 'flex', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', overflow: 'hidden' }}>
      {LANGUAGES.map((lang, i) => {
        const active = lang.code === value;
        return (
          <button
            key={lang.code}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={lang.label}
            onClick={() => {
              setInternal(lang.code);
              onChange?.(lang.code);
            }}
            style={{
              padding: '7px 11px',
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              fontWeight: 500,
              background: active ? 'var(--gray-950)' : 'transparent',
              color: active ? 'var(--gray-0)' : 'var(--text-primary)',
              border: 'none',
              borderLeft: i > 0 ? '1.5px solid var(--border-strong)' : 'none',
              cursor: 'pointer',
            }}
          >
            {lang.code}
          </button>
        );
      })}
    </div>
  );
}
