import React from 'react';

const STEPS = [
  { n: 1, label: 'Eligibility & class' },
  { n: 2, label: 'Applicant details' },
  { n: 3, label: 'Medical & documents' },
  { n: 4, label: 'Appointment, review & pay' },
];

export function ApplicationStepper({ current, onJump }: { current: number; onJump: (step: number) => void }) {
  return (
    <nav aria-label="Application progress" style={{ display: 'flex', borderBottom: '1px solid var(--border-default)' }}>
      {STEPS.map((s) => {
        const done = s.n < current;
        const active = s.n === current;
        const clickable = done;
        const state = active ? 'current' : done ? 'complete' : 'not started';
        return (
          <button
            key={s.n}
            type="button"
            disabled={!clickable}
            aria-current={active ? 'step' : undefined}
            onClick={() => clickable && onJump(s.n)}
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--space-3)',
              padding: 'var(--space-3) var(--space-4)',
              borderRight: '1px dashed var(--border-default)',
              background: active ? 'var(--surface-sunken)' : 'transparent',
              border: 'none',
              borderRadius: 0,
              cursor: clickable ? 'pointer' : 'default',
              textAlign: 'left',
            }}
          >
            <span
              aria-hidden="true"
              style={{
                flexShrink: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: 26,
                height: 26,
                borderRadius: '50%',
                border: '1.5px solid var(--border-strong)',
                background: active || done ? 'var(--gray-950)' : 'var(--surface-card)',
                color: active || done ? 'var(--gray-0)' : 'var(--text-primary)',
                font: '600 12px var(--font-mono)',
              }}
            >
              {done ? '✓' : s.n}
            </span>
            <span style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <span style={{ font: '600 12.5px var(--font-sans)' }}>{s.label}</span>
              <span style={{ font: '400 10.5px var(--font-mono)', color: 'var(--text-secondary)' }}>{state}</span>
            </span>
          </button>
        );
      })}
    </nav>
  );
}
