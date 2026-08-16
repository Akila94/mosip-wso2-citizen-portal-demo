import React, { useState } from 'react';
import { ExternalIdpShell } from '../components/identity/ExternalIdpShell';
import { RadioOption } from '../components/identity/RadioOption';
import { Input } from '../design-system/components/forms/Input';
import { Button } from '../design-system/components/core/Button';
import { Alert } from '../design-system/components/feedback/Alert';

export interface FederatedIdpScreenProps {
  onVerified: () => void;
}

export function FederatedIdpScreen({ onVerified }: FederatedIdpScreenProps) {
  const [method, setMethod] = useState<'push' | 'card'>('push');
  const [nic, setNic] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function handleVerify() {
    setSubmitting(true);
    // Simulated verification handshake with the external eID provider.
    await new Promise((r) => setTimeout(r, 500));
    setSubmitting(false);
    onVerified();
  }

  return (
    <ExternalIdpShell>
      <div style={{ padding: 'var(--space-8) var(--space-6)', display: 'flex', flexDirection: 'column', gap: 'var(--space-5)', maxWidth: 480, margin: '0 auto' }}>
        <h1 style={{ margin: 0, font: '700 22px var(--font-display)' }}>Authenticate with your national eID</h1>
        <Alert tone="info">
          A service operated by <strong>Marolia Digital Identity</strong> has asked to verify who you are. Only the attributes you approve will be released.
        </Alert>
        <Input label="NIC number" placeholder="enter your NIC" value={nic} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNic(e.target.value)} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          <span style={{ font: '600 12.5px var(--font-sans)' }}>Verification method</span>
          <RadioOption name="verify-method" label="eID mobile app push" checked={method === 'push'} onChange={() => setMethod('push')} />
          <RadioOption name="verify-method" label="Smart card reader" checked={method === 'card'} onChange={() => setMethod('card')} />
        </div>
        <Button fullWidth onClick={handleVerify} disabled={submitting}>{submitting ? 'Verifying…' : 'Verify and return'}</Button>
        <p style={{ margin: 0, font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
          ⚡ Federated authentication request sent; verified claims will be returned to WSO2 Identity Server and JIT-provisioned into the local user store.
        </p>
      </div>
    </ExternalIdpShell>
  );
}
