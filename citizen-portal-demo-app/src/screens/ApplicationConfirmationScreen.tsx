import React, { useEffect, useState } from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { Button } from '../design-system/components/core/Button';
import { Alert } from '../design-system/components/feedback/Alert';
import { applicationService } from '../services/applicationService';
import type { ApplicationConfirmation } from '../services/types';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

export function ApplicationConfirmationScreen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const [confirmation, setConfirmation] = useState<ApplicationConfirmation | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Submitting on mount is the wireframe's own flow (arriving here *is* the
  // submission). Now that this is a real, CSRF-protected POST it can fail, so
  // the rejection is handled — an unhandled one would leave this screen saying
  // "Submitting your application…" forever.
  //
  // Known gap: a browser refresh on this URL submits again. Making that safe
  // needs an idempotency key carried from the wizard, which is a change to the
  // submission contract rather than to this screen.
  useEffect(() => {
    let cancelled = false;
    applicationService
      .submitApplication({})
      .then((result) => {
        if (!cancelled) setConfirmation(result);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message || 'Could not submit your application.');
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <>
      <AppHeader screen="applicationConfirmation" onNavigate={onNavigate} breadcrumb={['Driving licence', 'Confirmation']} />
      <main className={styles.page} style={{ maxWidth: 700 }}>
        {error ? (
          <Alert tone="danger" title="We couldn't submit your application">
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
              <span>{error}</span>
              <Button variant="secondary" size="sm" onClick={() => onNavigate('appStep4')} style={{ alignSelf: 'flex-start' }}>
                Back to your application
              </Button>
            </div>
          </Alert>
        ) : !confirmation ? (
          <p style={{ font: '400 14px var(--font-sans)', color: 'var(--text-secondary)' }}>Submitting your application…</p>
        ) : (
          <>
            <div style={{ display: 'flex', gap: 'var(--space-4)', alignItems: 'flex-start' }}>
              <div style={{ flexShrink: 0, width: 38, height: 38, border: '2px solid var(--border-strong)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', font: '600 17px var(--font-mono)' }}>✓</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <h1 style={{ margin: 0, font: '700 22px var(--font-display)' }}>Application submitted</h1>
                <p style={{ margin: 0, font: '400 12.5px/1.55 var(--font-sans)', color: 'var(--text-secondary)' }}>We have your application and your test slot. Nothing further is needed until your appointment.</p>
              </div>
            </div>

            <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)' }}>
              {[
                ['Application reference', confirmation.reference],
                ['Payment reference', confirmation.paymentReference],
                ['Amount due', confirmation.amountDue],
                ['Appointment', confirmation.appointment],
                ['Location', confirmation.location],
                ['Expected processing', confirmation.processingEstimate],
              ].map(([k, v]) => (
                <div key={k} style={{ display: 'grid', gridTemplateColumns: '180px 1fr', gap: 'var(--space-3)', padding: 'var(--space-3) var(--space-4)', borderBottom: '1px dashed var(--border-default)' }}>
                  <span style={{ font: '400 12px var(--font-sans)', color: 'var(--text-secondary)' }}>{k}</span>
                  <span style={{ font: '600 12.5px var(--font-mono)' }}>{v}</span>
                </div>
              ))}
            </div>

            <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', background: 'var(--surface-sunken)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
              <span style={{ font: '600 13.5px var(--font-sans)' }}>What happens next</span>
              {confirmation.nextSteps.map((s, i) => (
                <span key={i} style={{ font: '400 12px/1.55 var(--font-sans)' }}>{s}</span>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 'var(--space-3)' }}>
              <Button fullWidth onClick={() => {}}>Download acknowledgement</Button>
              <Button variant="secondary" fullWidth onClick={() => onNavigate('landing')}>Back to portal</Button>
            </div>
          </>
        )}
      </main>
      <AppFooter />
    </>
  );
}
