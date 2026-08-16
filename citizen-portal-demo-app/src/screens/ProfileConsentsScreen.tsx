import React from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { AttributesTable } from '../components/profile/AttributesTable';
import { DepartmentRecordCard } from '../components/profile/DepartmentRecordCard';
import { ConsentRow } from '../components/profile/ConsentRow';
import { AsyncState } from '../components/common/AsyncState';
import { useProfileData } from '../hooks/usePortalData';
import { portalService } from '../services/portalService';
import { Alert } from '../design-system/components/feedback/Alert';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

export function ProfileConsentsScreen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const { attributes, records, consents } = useProfileData();

  const handleRevoke = async (id: string) => {
    await portalService.revokeConsent(id);
    consents.reload();
  };

  return (
    <>
      <AppHeader screen="profile" onNavigate={onNavigate} breadcrumb={['Profile & consents']} />
      <main className={styles.page}>
        <section aria-labelledby="attributes-heading" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <h2 id="attributes-heading" style={{ margin: 0, font: '700 20px var(--font-display)' }}>Identity attributes</h2>
          <AsyncState loading={attributes.loading} error={attributes.error} onRetry={attributes.reload} isEmpty={attributes.data?.length === 0}>
            <AttributesTable attributes={attributes.data ?? []} />
          </AsyncState>
          <Alert tone="info" title="How to read the source column">
            Verified attributes came from the identity provider and are read-only. Self-asserted attributes were typed by the citizen and can be edited.
          </Alert>
        </section>

        <section aria-labelledby="records-heading" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <h2 id="records-heading" style={{ margin: 0, font: '700 20px var(--font-display)' }}>Your records across government</h2>
          <AsyncState loading={records.loading} error={records.error} onRetry={records.reload} isEmpty={records.data?.length === 0}>
            <ul className={styles.recordGrid} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              {(records.data ?? []).map((r) => (
                <li key={r.id}>
                  <DepartmentRecordCard record={r} />
                </li>
              ))}
            </ul>
          </AsyncState>
        </section>

        <section aria-labelledby="consents-heading" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <h2 id="consents-heading" style={{ margin: 0, font: '700 20px var(--font-display)' }}>Micro apps with access to your data</h2>
          <AsyncState loading={consents.loading} error={consents.error} onRetry={consents.reload} isEmpty={consents.data?.length === 0} emptyMessage="No apps currently have access to your data.">
            <div style={{ border: '1.5px solid var(--border-default)', borderRadius: 'var(--radius-md)', overflow: 'hidden' }}>
              {(consents.data ?? []).map((c) => (
                <ConsentRow key={c.id} consent={c} onRevoke={handleRevoke} />
              ))}
            </div>
          </AsyncState>
        </section>
      </main>
      <AppFooter />
    </>
  );
}
