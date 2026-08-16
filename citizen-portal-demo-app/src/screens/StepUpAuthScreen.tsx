import React, { useState } from 'react';
import { IdentityShell } from '../components/identity/IdentityShell';
import { RadioOption } from '../components/identity/RadioOption';
import { OtpInput } from '../components/identity/OtpInput';
import { identityService } from '../services/identityService';
import { useAuth } from '../context/AuthContext';
import { Button } from '../design-system/components/core/Button';
import { Alert } from '../design-system/components/feedback/Alert';
import { Badge } from '../design-system/components/core/Badge';

export function StepUpAuthScreen({ onConfirmed, onUseAnotherMethod }: { onConfirmed: () => void; onUseAnotherMethod: () => void }) {
  const { raiseAssurance } = useAuth();
  const [method, setMethod] = useState<'totp' | 'sms'>('totp');
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleConfirm() {
    setSubmitting(true);
    setError(null);
    try {
      const result = await identityService.submitStepUp(code);
      raiseAssurance(result.assuranceLevel);
      onConfirmed();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <IdentityShell>
      <div style={{ padding: 'var(--space-8) var(--space-6)', maxWidth: 520, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 'var(--space-5)' }}>
        <Badge tone="warning">Adaptive step-up · ACR raised to substantial</Badge>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-1)' }}>
          <h1 style={{ margin: 0, font: '700 22px var(--font-display)' }}>One more check before you submit and pay</h1>
          <p style={{ margin: 0, font: '400 12.5px/1.55 var(--font-sans)', color: 'var(--text-secondary)' }}>
            You are already signed in. Because this step submits a legal declaration and takes a payment, the service asked us to confirm it is really you.
          </p>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          <RadioOption name="stepup-method" label="Authenticator app code (TOTP)" hint="6 digits, refreshes every 30s" checked={method === 'totp'} onChange={() => setMethod('totp')} />
          <RadioOption name="stepup-method" label="SMS one-time code" hint="to +94 7•• ••• 220" checked={method === 'sms'} onChange={() => setMethod('sms')} />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          <span style={{ font: '600 12.5px var(--font-sans)' }}>Enter your 6-digit code</span>
          <OtpInput value={code} onChange={setCode} />
        </div>
        {error && <Alert tone="danger">{error}</Alert>}
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <Button fullWidth onClick={handleConfirm} disabled={submitting}>{submitting ? 'Confirming…' : 'Confirm'}</Button>
          <Button variant="secondary" onClick={onUseAnotherMethod}>Use another method</Button>
        </div>
        <p style={{ margin: 0, font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
          ⚡ The micro app requested a higher assurance level; the identity server escalates and returns an upgraded session — no re-entry of the password.
        </p>
      </div>
    </IdentityShell>
  );
}
