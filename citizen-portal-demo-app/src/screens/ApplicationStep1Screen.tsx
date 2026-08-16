import React from 'react';
import { MicroAppHeader } from '../components/application/MicroAppHeader';
import { ApplicationStepper } from '../components/application/ApplicationStepper';
import { VerifiedIdentityPanel } from '../components/application/VerifiedIdentityPanel';
import { SelectableTile } from '../components/application/SelectableTile';
import { ApplicationFooterNav } from '../components/application/ApplicationFooterNav';
import { AsyncState } from '../components/common/AsyncState';
import { AppFooter } from '../components/layout/AppFooter';
import { useApplicationConfig } from '../hooks/useApplicationData';
import type { LicenceApplicationWizardState } from '../hooks/useLicenceApplicationWizard';
import styles from '../styles/layout.module.css';

export interface ApplicationStepProps {
  wizard: LicenceApplicationWizardState;
  onBack: () => void;
  onContinue: () => void;
  onJumpStep: (step: number) => void;
  onSaveExit: () => void;
}

export function ApplicationStep1Screen({ wizard, onBack, onContinue, onJumpStep, onSaveExit }: ApplicationStepProps) {
  const { data: config, loading, error, reload } = useApplicationConfig();

  return (
    <>
      <MicroAppHeader onSaveExit={onSaveExit} appName="Driving Licence Service" tagline="micro app · Dept. of Motor Traffic · within MaroliaGov" />
      <ApplicationStepper current={1} onJump={onJumpStep} />
      <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading application…">
        {config && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div className={styles.page} style={{ flex: 1, margin: 0, maxWidth: 'none' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Eligibility &amp; licence class</h1>
                <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>Fee $20 · 10 working days · you can save and return at any point.</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <span style={{ font: '600 13.5px var(--font-sans)' }}>What are you applying for?</span>
                <div style={{ display: 'flex', gap: 'var(--space-3)' }}>
                  {config.appTypes.map((t) => (
                    <SelectableTile key={t.id} type="radio" name="app-type" label={t.label} description={t.description} selected={wizard.appTypeId === t.id} onSelect={() => wizard.setAppTypeId(t.id)} />
                  ))}
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
                  <span style={{ font: '600 13.5px var(--font-sans)' }}>Licence classes</span>
                  <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>select all that apply · fee is per application, not per class</span>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 'var(--space-3)' }}>
                  {config.licenceClasses.map((c) => (
                    <SelectableTile
                      key={c.id}
                      type="checkbox"
                      label={c.label}
                      description={c.description}
                      selected={wizard.selectedClassIds.includes(c.id)}
                      onSelect={() => wizard.toggleClass(c.id)}
                      disabled={!c.eligible && !wizard.selectedClassIds.includes(c.id)}
                      disabledReason={c.ineligibleReason}
                    />
                  ))}
                </div>
                {config.licenceClasses.filter((c) => !c.eligible && wizard.selectedClassIds.includes(c.id)).map((c) => (
                  <div key={c.id} role="alert" style={{ border: '1px solid var(--warning)', background: 'var(--surface-sunken)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }} aria-live="polite">
                    <span style={{ font: '600 13px var(--font-sans)' }}>Class {c.label.split(' — ')[0]} cannot be included</span>
                    <span style={{ font: '400 12px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{c.ineligibleReason}</span>
                    <button type="button" onClick={() => wizard.toggleClass(c.id)} style={{ background: 'none', border: 'none', padding: 0, alignSelf: 'flex-start', font: '600 12px var(--font-sans)', textDecoration: 'underline', cursor: 'pointer' }}>
                      Remove {c.label.split(' — ')[0]} and continue
                    </button>
                  </div>
                ))}
              </div>

              <div style={{ display: 'flex', gap: 'var(--space-5)' }}>
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <span style={{ font: '600 13px var(--font-sans)' }}>Learner permit number</span>
                  <div style={{ border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', font: '400 13px var(--font-mono)' }}>{config.permitNumber}</div>
                  <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>Format LP-YYYY-NNNNN. Label stays visible above the field.</span>
                </div>
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <span style={{ font: '600 13px var(--font-sans)' }}>Permit issue date</span>
                  <div style={{ border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', font: '400 13px var(--font-mono)' }}>{config.permitIssueDate}</div>
                  <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>Must be at least 30 days ago.</span>
                </div>
              </div>

              <div role="status" aria-live="polite" style={{ border: '1px solid var(--success)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)', background: 'var(--surface-sunken)' }}>
                <span style={{ font: '600 14px var(--font-sans)' }}>✓ You are eligible for class B (motor car)</span>
                <span style={{ font: '400 12px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>— You are 30 years old — above the minimum of 18 for class B.</span>
                <span style={{ font: '400 12px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>— Learner permit {config.permitNumber} was issued 133 days ago — above the 30-day minimum.</span>
                <span style={{ font: '400 12px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>— No active suspension found against your verified identity record.</span>
              </div>

              <ApplicationFooterNav backLabel="Back to service page" onBack={onBack} continueLabel="Continue to step 2" onContinue={onContinue} autoSaveNote />
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%' }}>
              <VerifiedIdentityPanel identity={config.verifiedIdentity} showSessionNote />
            </div>
          </div>
        )}
      </AsyncState>
      <AppFooter compact />
    </>
  );
}
