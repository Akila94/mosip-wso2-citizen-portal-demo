import React from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { AsyncState } from '../components/common/AsyncState';
import { useMedicalReviewError } from '../hooks/useApplicationData';
import { Button } from '../design-system/components/core/Button';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

/** Shared failure-explanation pattern (frame 18): name the rule, name the
 * reference, offer one concrete next action. Reused for any application
 * failure — the medical-review case is the concrete demo path, reached by
 * declaring blackouts in step 3, per product decision. */
export function ApplicationErrorScreen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const { data: info, loading, error, reload } = useMedicalReviewError();

  return (
    <>
      <AppHeader screen="applicationError" onNavigate={onNavigate} breadcrumb={['Driving licence', 'Medical review needed']} />
      <main className={styles.page} style={{ maxWidth: 700 }}>
        <AsyncState loading={loading} error={error} onRetry={reload}>
          {info && (
            <>
              <div style={{ display: 'flex', gap: 'var(--space-4)', alignItems: 'flex-start' }}>
                <div style={{ flexShrink: 0, width: 38, height: 38, border: '2px solid var(--border-strong)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', font: '600 17px var(--font-mono)' }}>!</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                  <h1 style={{ margin: 0, font: '700 20px/1.3 var(--font-display)' }}>{info.title}</h1>
                  <p style={{ margin: 0, font: '400 12.5px/1.55 var(--font-sans)', color: 'var(--text-secondary)' }}>{info.description}</p>
                </div>
              </div>

              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>{info.reasonHeading}</span>
                <p style={{ margin: 0, font: '400 12.5px/1.6 var(--font-sans)' }}>{info.reasonBody}</p>
                <div style={{ borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>{info.reference}</div>
              </div>

              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', background: 'var(--surface-sunken)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <span style={{ font: '600 13.5px var(--font-sans)' }}>Your route forward</span>
                {info.routes.map((r, i) => (
                  <div key={i} style={{ display: 'flex', gap: 'var(--space-3)', borderTop: i > 0 ? '1px dashed var(--border-default)' : 'none', paddingTop: i > 0 ? 'var(--space-3)' : 0 }}>
                    <span style={{ font: '600 12px var(--font-mono)' }}>{i + 1}</span>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      <span style={{ font: '600 12.5px var(--font-sans)' }}>{r.label}</span>
                      <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{r.description}</span>
                    </div>
                  </div>
                ))}
              </div>

              <div style={{ display: 'flex', gap: 'var(--space-3)' }}>
                <Button fullWidth onClick={() => {}}>Book a medical review</Button>
                <Button variant="secondary" fullWidth onClick={() => onNavigate('landing')}>Save and exit</Button>
              </div>
            </>
          )}
        </AsyncState>
      </main>
      <AppFooter />
    </>
  );
}
