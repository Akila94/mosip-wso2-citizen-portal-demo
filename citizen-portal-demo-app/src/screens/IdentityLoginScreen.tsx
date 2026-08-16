import React, { useState } from 'react';
import { IdentityShell } from '../components/identity/IdentityShell';
import { FederatedProviderRow } from '../components/identity/FederatedProviderRow';
import { AsyncState } from '../components/common/AsyncState';
import { useIdentityProviders } from '../hooks/useIdentityData';
import { useServiceDetail } from '../hooks/useIdentityData';
import { identityService } from '../services/identityService';
import { Input } from '../design-system/components/forms/Input';
import { Tabs } from '../design-system/components/navigation/Tabs';
import { Button } from '../design-system/components/core/Button';
import { Alert } from '../design-system/components/feedback/Alert';

export interface IdentityLoginScreenProps {
  serviceId: string;
  onLocalSignIn: () => void;
  onFederatedSelect: (idpId: string, externalHop: boolean) => void;
  onCancel: () => void;
}

export function IdentityLoginScreen({ serviceId, onLocalSignIn, onFederatedSelect, onCancel }: IdentityLoginScreenProps) {
  const { data: service } = useServiceDetail(serviceId);
  const { data: idps, loading, error, reload } = useIdentityProviders();
  const [identifierType, setIdentifierType] = useState<'email' | 'mobile'>('email');
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  async function handleLocalSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!identifier || !password) {
      setFormError(`Enter your ${identifierType === 'email' ? 'email address' : 'mobile number'} and password.`);
      return;
    }
    setFormError(null);
    setSubmitting(true);
    try {
      await identityService.signInLocal(identifier, password);
      onLocalSignIn();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Sign-in failed.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <IdentityShell serviceName={service?.title ?? 'Driving Licence Service'}>
      <div style={{ padding: 'var(--space-8) var(--space-6)', display: 'flex', justifyContent: 'center', gap: 'var(--space-8)', flexWrap: 'wrap' }}>
        <form onSubmit={handleLocalSubmit} style={{ width: 400, display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }} noValidate>
          <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Sign in</h1>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-1)' }}>
            <span style={{ font: '600 12.5px var(--font-sans)' }}>Sign in with</span>
            <Tabs tabs={[{ value: 'email', label: 'Email' }, { value: 'mobile', label: 'Mobile number' }]} value={identifierType} onChange={(v) => setIdentifierType(v as 'email' | 'mobile')} />
          </div>
          <Input
            label={identifierType === 'email' ? 'Email address' : 'Mobile number'}
            type="text"
            inputMode={identifierType === 'email' ? 'email' : 'tel'}
            placeholder={identifierType === 'email' ? 'you@example.mr' : '+94 7•• ••• •••'}
            value={identifier}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setIdentifier(e.target.value)}
            name="mg-identifier"
            autoComplete="off"
            autoCorrect="off"
            autoCapitalize="off"
            spellCheck={false}
            data-lpignore="true"
            data-1p-ignore="true"
          />
          <Input
            label="Password"
            type={showPassword ? 'text' : 'password'}
            placeholder="••••••••"
            value={password}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
            autoComplete="current-password"
            hint={
              <button type="button" onClick={() => setShowPassword((v) => !v)} style={{ background: 'none', border: 'none', padding: 0, font: '600 11px var(--font-mono)', color: 'var(--text-link)', cursor: 'pointer' }}>
                {showPassword ? 'hide' : 'show'}
              </button>
            }
          />
          {formError && <Alert tone="danger">{formError}</Alert>}
          <Button type="submit" fullWidth disabled={submitting}>{submitting ? 'Signing in…' : 'Sign in'}</Button>
          <div style={{ display: 'flex', justifyContent: 'space-between', font: '600 12px var(--font-sans)' }}>
            <a href="#forgot-password">Forgot password?</a>
            <a href="#create-account">Create an account</a>
          </div>
          <p style={{ margin: 0, font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
            Self-registration creates a Basic account — assurance is raised later by federating with National Digital ID.
          </p>
          <button type="button" onClick={onCancel} style={{ background: 'none', border: 'none', padding: 0, font: '500 12px var(--font-sans)', color: 'var(--text-secondary)', textAlign: 'left', cursor: 'pointer' }}>
            ← Cancel and return to service
          </button>
        </form>

        <div style={{ width: 1.5, background: 'var(--border-default)', alignSelf: 'stretch' }} />

        <div style={{ width: 400, display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <h2 style={{ margin: 0, font: '600 15px var(--font-sans)' }}>Or continue with</h2>
          <AsyncState loading={loading} error={error} onRetry={reload}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
              {(idps ?? []).map((p) => (
                <FederatedProviderRow key={p.id} provider={p} onClick={() => onFederatedSelect(p.id, p.externalHop)} />
              ))}
            </div>
          </AsyncState>
          <Alert tone="info" title="Assurance differs by method">
            National Digital ID or MOSIP → substantial (verified account). Mobile OTP or password alone → basic; a service needing substantial will step up or ask to federate.
          </Alert>
        </div>
      </div>
    </IdentityShell>
  );
}
