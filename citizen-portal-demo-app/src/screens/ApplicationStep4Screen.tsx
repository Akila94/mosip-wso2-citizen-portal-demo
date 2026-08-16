import React from 'react';
import { MicroAppHeader } from '../components/application/MicroAppHeader';
import { ApplicationStepper } from '../components/application/ApplicationStepper';
import { VerifiedIdentityPanel } from '../components/application/VerifiedIdentityPanel';
import { TestSlotGrid } from '../components/application/TestSlotGrid';
import { ApplicationReview } from '../components/application/ApplicationReview';
import { AsyncState } from '../components/common/AsyncState';
import { AppFooter } from '../components/layout/AppFooter';
import { useApplicationConfig } from '../hooks/useApplicationData';
import { useTestSlotBooking } from '../hooks/useTestSlotBooking';
import { Checkbox } from '../design-system/components/forms/Checkbox';
import { Button } from '../design-system/components/core/Button';
import type { ApplicationStepProps } from './ApplicationStep1Screen';
import styles from '../styles/layout.module.css';

export interface ApplicationStep4ScreenProps extends Omit<ApplicationStepProps, 'onContinue'> {
  onSubmit: () => void;
}

export function ApplicationStep4Screen({ wizard, onBack, onJumpStep, onSaveExit, onSubmit }: ApplicationStep4ScreenProps) {
  const { data: config, loading, error, reload } = useApplicationConfig();
  const booking = useTestSlotBooking();

  return (
    <>
      <MicroAppHeader onSaveExit={onSaveExit} appName="Driving Licence Service" tagline="micro app · Dept. of Motor Traffic · within MaroliaGov" />
      <ApplicationStepper current={4} onJump={onJumpStep} />
      <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading application…">
        {config && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div className={styles.page} style={{ flex: 1, margin: 0, maxWidth: 'none' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
                  <h2 style={{ margin: 0, font: '600 16px var(--font-sans)' }}>Book your written test</h2>
                  <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>{wizard.pickupStation} · from step 2</span>
                </div>
                <AsyncState loading={booking.loading} error={booking.error} onRetry={booking.reload} loadingLabel="Loading test slots…">
                  <TestSlotGrid
                    week={booking.week}
                    weekLabel={booking.weekLabel}
                    selected={booking.selected}
                    countdownLabel={booking.countdownLabel}
                    secondsRemaining={booking.secondsRemaining}
                    office={wizard.pickupStation}
                    onSelect={booking.selectSlot}
                    onPrevWeek={() => booking.changeWeek(-1)}
                    onNextWeek={() => booking.changeWeek(1)}
                    canGoPrev={booking.weekOffset > 0}
                  />
                </AsyncState>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <h2 style={{ margin: 0, font: '600 16px var(--font-sans)' }}>Review your application</h2>
                <ApplicationReview sections={config.review} identity={config.verifiedIdentity} />
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                  <span style={{ font: '600 15px var(--font-sans)' }}>Fee breakdown</span>
                  {config.feeBreakdown.map((f) => (
                    <div key={f.label} style={{ display: 'flex', justifyContent: 'space-between', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 12.5px var(--font-sans)' }}>
                      <span>{f.label}</span>
                      <span style={{ fontWeight: 600 }}>{f.amount}</span>
                    </div>
                  ))}
                  <div style={{ display: 'flex', justifyContent: 'space-between', borderTop: '1.5px solid var(--border-strong)', paddingTop: 'var(--space-3)' }}>
                    <span style={{ font: '600 14px var(--font-sans)' }}>Total</span>
                    <span style={{ font: '600 20px var(--font-sans)' }}>{config.totalFee}</span>
                  </div>
                </div>
                <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                  <span style={{ font: '600 15px var(--font-sans)' }}>How to pay</span>
                  {(['now', 'later'] as const).map((v) => (
                    <label key={v} style={{ border: `1.5px solid ${wizard.paymentMethod === v ? 'var(--accent)' : 'var(--border-strong)'}`, borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', display: 'flex', flexDirection: 'column', gap: 6, cursor: 'pointer' }}>
                      <span style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)' }}>
                        <input type="radio" name="payment" checked={wizard.paymentMethod === v} onChange={() => wizard.setPaymentMethod(v)} style={{ accentColor: 'var(--accent)' }} />
                        <span style={{ font: '600 13px var(--font-sans)' }}>{v === 'now' ? 'Pay now' : 'Pay later with a reference'}</span>
                      </span>
                      <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
                        {v === 'now' ? 'Card or mobile wallet. Confirmation is immediate.' : 'Submitting generates a payment reference valid 7 days, payable at any bank, post office, agent or by USSD. Your slot is held for 48 hours.'}
                      </span>
                    </label>
                  ))}
                  <span style={{ font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>payment is decoupled from submission — the application is lodged either way</span>
                </div>
              </div>

              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
                <Checkbox
                  label="I declare that the information given is true and complete, and that I understand a false declaration is an offence under the Motor Traffic Act."
                  checked={wizard.declarationAgreed}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => wizard.setDeclarationAgreed(e.target.checked)}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-4)' }}>
                  <Button variant="secondary" onClick={() => wizard.setDigitalSignature(!wizard.digitalSignature)}>
                    {wizard.digitalSignature ? 'Digital certificate selected ✓' : 'Sign with your digital certificate'}
                  </Button>
                  <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)', flex: 1 }}>
                    Optional for this service. Required for vehicle transfer and tax filing — the same certificate, held in your national eID.
                  </span>
                </div>
              </div>

              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1.5px solid var(--border-default)', paddingTop: 'var(--space-5)' }}>
                <Button variant="secondary" onClick={onBack}>Back to step 3</Button>
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
                  {!booking.selected && <span style={{ font: '500 11px var(--font-mono)', color: 'var(--danger)' }}>hold a test slot above to continue</span>}
                  <span style={{ font: '500 11px var(--font-mono)', color: 'var(--text-secondary)' }}>→ triggers step-up verification</span>
                  <Button onClick={onSubmit} disabled={!wizard.declarationAgreed || !booking.selected}>Submit application</Button>
                </div>
              </div>
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
              <VerifiedIdentityPanel identity={config.verifiedIdentity} />
              <div style={{ border: '1px solid var(--border-default)', background: 'var(--surface-card)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Before you submit</span>
                <span style={{ font: '400 11.5px/1.55 var(--font-sans)' }}>This step submits a legal declaration and takes a payment, so the identity service will ask for one extra check — a code from your authenticator app. You will not re-enter your password.</span>
              </div>
            </div>
          </div>
        )}
      </AsyncState>
      <AppFooter compact />
    </>
  );
}
