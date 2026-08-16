import React, { useState } from 'react';
import { SessionExpiredModal } from '../components/identity/SessionExpiredModal';
import { ApplicationStep2Screen } from './ApplicationStep2Screen';
import { useLicenceApplicationWizard } from '../hooks/useLicenceApplicationWizard';

/** Demo entry for frame 12 — now wired against the real frame 14 (Step 2)
 * from Document C, per the wireframe's own note that the modal fires over
 * an in-progress application step. The underlying step is real but frozen
 * (pointer-events disabled, aria-hidden) while the modal is up. */
export function SessionExpiredDemoScreen({ onStartNewSession, onBackToPortal }: { onStartNewSession: () => void; onBackToPortal: () => void }) {
  const [dismissed, setDismissed] = useState(false);
  const wizard = useLicenceApplicationWizard();

  return (
    <div style={{ minHeight: '100vh' }}>
      <div style={{ filter: dismissed ? 'none' : 'grayscale(1)', opacity: dismissed ? 1 : 0.4, pointerEvents: dismissed ? 'auto' : 'none' }} aria-hidden={!dismissed}>
        <ApplicationStep2Screen wizard={wizard} onBack={() => {}} onContinue={() => {}} onJumpStep={() => {}} onSaveExit={() => {}} />
      </div>
      {!dismissed && (
        <SessionExpiredModal
          onStartNewSession={() => {
            setDismissed(true);
            onStartNewSession();
          }}
          onBackToPortal={onBackToPortal}
        />
      )}
    </div>
  );
}
