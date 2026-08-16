import React from 'react';
import type { DeclarationAnswer } from '../../services/types';

/** Deviation: the design system has no two-option segmented/pill control
 * (Switch is boolean-styled as a toggle, not a labelled Yes/No pair). Built
 * from DS tokens as a two-button group, consistent with the language
 * switcher's segmented-control pattern used elsewhere in this app. */
export function DeclarationRow({ question, help, answer, onAnswer }: { question: string; help: string; answer: DeclarationAnswer; onAnswer: (a: DeclarationAnswer) => void }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 'var(--space-4)', alignItems: 'center', padding: 'var(--space-4)', borderTop: '1px dashed var(--border-default)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        <span role="group" aria-label={question} style={{ font: '600 13.5px var(--font-sans)' }}>{question}</span>
        <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{help}</span>
      </div>
      <div style={{ display: 'flex', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-pill)', overflow: 'hidden', justifySelf: 'end' }}>
        {(['yes', 'no'] as const).map((v, i) => {
          const active = answer === v;
          return (
            <button
              key={v}
              type="button"
              aria-pressed={active}
              onClick={() => onAnswer(v)}
              style={{
                padding: '9px 18px',
                font: '600 12px var(--font-mono)',
                border: 'none',
                borderLeft: i > 0 ? '1.5px solid var(--border-strong)' : 'none',
                background: active ? 'var(--gray-950)' : 'transparent',
                color: active ? 'var(--gray-0)' : 'var(--text-secondary)',
                cursor: 'pointer',
                textTransform: 'capitalize',
              }}
            >
              {v}
            </button>
          );
        })}
      </div>
    </div>
  );
}
