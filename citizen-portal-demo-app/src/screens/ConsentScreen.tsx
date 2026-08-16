import React, { useEffect, useState } from 'react';
import { IdentityShell } from '../components/identity/IdentityShell';
import { ConsentAttributeRow } from '../components/identity/ConsentAttributeRow';
import { AsyncState } from '../components/common/AsyncState';
import { useConsentAttributes, useServiceDetail } from '../hooks/useIdentityData';
import { identityService } from '../services/identityService';
import { Button } from '../design-system/components/core/Button';
import { Alert } from '../design-system/components/feedback/Alert';

export interface ConsentScreenProps {
  serviceId: string;
  onAllow: () => void;
  onDeny: () => void;
}

export function ConsentScreen({ serviceId, onAllow, onDeny }: ConsentScreenProps) {
  const { data: service } = useServiceDetail(serviceId);
  const { data: attributes, loading, error, reload } = useConsentAttributes(serviceId);
  const [decisions, setDecisions] = useState<Record<string, boolean>>({});
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (attributes) setDecisions(Object.fromEntries(attributes.map((a) => [a.id, true])));
  }, [attributes]);

  async function handleAllow() {
    setSubmitting(true);
    await identityService.submitConsent(serviceId, decisions);
    setSubmitting(false);
    onAllow();
  }

  return (
    <IdentityShell serviceName={service?.title ?? 'Driving Licence Service'}>
      <div style={{ padding: 'var(--space-8) var(--space-6)', maxWidth: 640, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 'var(--space-5)' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-1)' }}>
          <h1 style={{ margin: 0, font: '700 22px/1.3 var(--font-display)' }}>{service?.title ?? 'This service'} is requesting your details</h1>
          <p style={{ margin: 0, font: '400 12.5px/1.55 var(--font-sans)', color: 'var(--text-secondary)' }}>
            Operated by {service?.department ?? 'the requesting agency'}. Choose what to share — required attributes are marked.
          </p>
        </div>

        <AsyncState loading={loading} error={error} onRetry={reload}>
          <div style={{ border: '1.5px solid var(--border-default)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
            {(attributes ?? []).map((a) => (
              <ConsentAttributeRow key={a.id} attribute={a} allowed={decisions[a.id] ?? true} onToggle={(v) => setDecisions((d) => ({ ...d, [a.id]: v }))} />
            ))}
          </div>
        </AsyncState>

        <Alert tone="info" title="Retention">
          The service keeps these attributes for the life of the application plus 6 years, as required by the Motor Traffic Act. You can revoke access at any time in Profile &amp; Consents.
        </Alert>

        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <Button fullWidth onClick={handleAllow} disabled={submitting}>{submitting ? 'Submitting…' : 'Allow and continue'}</Button>
          <Button variant="secondary" onClick={onDeny}>Deny</Button>
        </div>
        <p style={{ margin: 0, font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
          Denying a required attribute would normally block the service (flagged: the blocking flow and its explanation screen belong to a later document).
        </p>
      </div>
    </IdentityShell>
  );
}
