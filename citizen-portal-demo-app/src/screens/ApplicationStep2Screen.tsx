import React, { useState } from 'react';
import { MicroAppHeader } from '../components/application/MicroAppHeader';
import { ApplicationStepper } from '../components/application/ApplicationStepper';
import { VerifiedIdentityPanel } from '../components/application/VerifiedIdentityPanel';
import { SelfAssertedPanel } from '../components/application/SelfAssertedPanel';
import { ApplicationFooterNav } from '../components/application/ApplicationFooterNav';
import { AsyncState } from '../components/common/AsyncState';
import { AppFooter } from '../components/layout/AppFooter';
import { SessionInspectorToggle } from '../components/session/SessionInspectorToggle';
import { SessionInspectorPanel } from '../components/session/SessionInspectorPanel';
import { useApplicationConfig } from '../hooks/useApplicationData';
import { Input } from '../design-system/components/forms/Input';
import { Select } from '../design-system/components/forms/Select';
import { Checkbox } from '../design-system/components/forms/Checkbox';
import { Alert } from '../design-system/components/feedback/Alert';
import type { ApplicationStepProps } from './ApplicationStep1Screen';
import styles from '../styles/layout.module.css';

export function ApplicationStep2Screen({ wizard, onBack, onContinue, onJumpStep, onSaveExit }: ApplicationStepProps) {
  const { data: config, loading, error, reload } = useApplicationConfig();
  const [inspectorOpen, setInspectorOpen] = useState(false);

  return (
    <>
      <MicroAppHeader onSaveExit={onSaveExit} appName="Driving Licence Service" tagline="micro app · Dept. of Motor Traffic · within MaroliaGov" />
      <ApplicationStepper current={2} onJump={onJumpStep} />
      <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading application…">
        {config && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div className={styles.page} style={{ flex: 1, margin: 0, maxWidth: 'none' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Applicant details</h1>
                <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>We only ask for what your identity record does not already tell us.</span>
              </div>

              <Alert tone="info">
                Name, NIC, date of birth, address and photograph are already verified — see the panel on the right. Nothing identity-derived is repeated as a greyed-out field here.
              </Alert>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-5)' }}>
                {config.editableFields.map((f) => (
                  <Input
                    key={f.id}
                    label={`${f.label}${f.required ? '' : ' (optional)'}`}
                    defaultValue={wizard.fieldValues[f.id] ?? f.value}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => wizard.setField(f.id, e.target.value)}
                    hint={f.help}
                  />
                ))}
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)' }}>
                <span style={{ font: '600 15px var(--font-sans)' }}>Where will you collect your licence?</span>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                  <Select label="Collection district" options={config.collectionDistricts} value={wizard.collectionDistrict} onChange={(e: React.ChangeEvent<HTMLSelectElement>) => wizard.setCollectionDistrict(e.target.value)} />
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    <Select label="Pick-up station" options={config.pickupStations} value={wizard.pickupStation} onChange={(e: React.ChangeEvent<HTMLSelectElement>) => wizard.setPickupStation(e.target.value)} />
                    <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>Stations are filtered by the district above. Test slots in step 4 follow this choice.</span>
                  </div>
                </div>
                <Checkbox label="Post it to my verified address instead (renewals only, +$3)" checked={wizard.postInstead} onChange={(e: React.ChangeEvent<HTMLInputElement>) => wizard.setPostInstead(e.target.checked)} />
              </div>

              <ApplicationFooterNav backLabel="Back to step 1" onBack={onBack} continueLabel="Continue to step 3" onContinue={onContinue} />
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
              <VerifiedIdentityPanel identity={config.verifiedIdentity} />
              <SelfAssertedPanel attributes={config.selfAsserted} />
            </div>
          </div>
        )}
      </AsyncState>
      <div style={{ display: 'flex', alignItems: 'center', padding: '0 var(--space-8)', background: 'var(--surface-sunken)' }}>
        <SessionInspectorToggle open={inspectorOpen} onToggle={() => setInspectorOpen((v) => !v)} />
      </div>
      {inspectorOpen && <SessionInspectorPanel clientId="driving-licence" onCollapse={() => setInspectorOpen(false)} />}
      <AppFooter compact />
    </>
  );
}
