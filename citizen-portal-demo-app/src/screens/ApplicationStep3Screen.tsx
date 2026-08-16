import React from 'react';
import { MicroAppHeader } from '../components/application/MicroAppHeader';
import { ApplicationStepper } from '../components/application/ApplicationStepper';
import { VerifiedIdentityPanel } from '../components/application/VerifiedIdentityPanel';
import { DeclarationRow } from '../components/application/DeclarationRow';
import { DocumentUploadCard } from '../components/application/DocumentUploadCard';
import { ApplicationFooterNav } from '../components/application/ApplicationFooterNav';
import { AsyncState } from '../components/common/AsyncState';
import { AppFooter } from '../components/layout/AppFooter';
import { useApplicationConfig } from '../hooks/useApplicationData';
import type { ApplicationStepProps } from './ApplicationStep1Screen';
import styles from '../styles/layout.module.css';

export function ApplicationStep3Screen({ wizard, onBack, onContinue, onJumpStep, onSaveExit }: ApplicationStepProps) {
  const { data: config, loading, error, reload } = useApplicationConfig();

  return (
    <>
      <MicroAppHeader onSaveExit={onSaveExit} appName="Driving Licence Service" tagline="micro app · Dept. of Motor Traffic · within MaroliaGov" />
      <ApplicationStepper current={3} onJump={onJumpStep} />
      <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading application…">
        {config && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div className={styles.page} style={{ flex: 1, margin: 0, maxWidth: 'none' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Medical declaration &amp; documents</h1>
                <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>A false declaration is an offence under the Motor Traffic Act.</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <span style={{ font: '600 15px var(--font-sans)' }}>Health declarations</span>
                <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)' }}>
                  {config.declarations.map((d) => (
                    <DeclarationRow key={d.id} question={d.question} help={d.help} answer={wizard.declarationAnswers[d.id] ?? null} onAnswer={(a) => wizard.setDeclaration(d.id, a)} />
                  ))}
                  <div style={{ padding: 'var(--space-4)', background: 'var(--surface-sunken)', font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
                    A "yes" to corrected vision adds a condition code to the licence rather than blocking it. A "yes" to epilepsy or blackouts routes to a medical review, with the reason and the route forward.
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
                  <span style={{ font: '600 15px var(--font-sans)' }}>Documents</span>
                  <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>specs are stated before you upload, not after a rejection</span>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                  <DocumentUploadCard title="Medical fitness certificate" spec="accepted · PDF or JPG · max 5 MB · issued within 3 months" file={wizard.medicalCert} onFileSelected={() => {}} required />
                  <DocumentUploadCard title="Learner permit" spec="accepted · PDF or JPG · max 5 MB · both sides in one file" file={wizard.learnerPermitDoc} onFileSelected={wizard.setLearnerPermitDoc} required />
                </div>
              </div>

              <ApplicationFooterNav backLabel="Back to step 2" onBack={onBack} continueLabel="Continue to step 4" onContinue={onContinue} />
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
              <VerifiedIdentityPanel identity={config.verifiedIdentity} />
              <div style={{ border: '1px dotted var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-primary)' }}>
                <strong>Note</strong> — uploads are stored by the micro app, not by the identity service. Nothing in this step crosses the identity trust boundary.
              </div>
            </div>
          </div>
        )}
      </AsyncState>
      <AppFooter compact />
    </>
  );
}
