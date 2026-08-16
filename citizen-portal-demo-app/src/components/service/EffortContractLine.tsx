import React from 'react';
import type { EffortContract } from '../../services/types';

function signatureLabel(sig: EffortContract['signature']) {
  if (sig === 'required') return 'e-signature required';
  if (sig === 'optional') return 'e-signature optional';
  return 'no signature';
}

/** Mandatory line on every service card, before any click: sign-in ·
 * signature · steps · time · fee. "No sign-in needed" is stated positively
 * where it applies, per the wireframe's a11y annotation. */
export function EffortContractLine({ effort, signedIn }: { effort: EffortContract; signedIn: boolean }) {
  const signInLabel = !effort.signInRequired ? 'no sign-in needed' : signedIn ? 'signed in ✓' : 'sign-in required';
  const parts = [signInLabel, signatureLabel(effort.signature), `${effort.steps} step${effort.steps > 1 ? 's' : ''}`, `~${effort.timeEstimate}`, effort.fee];
  return (
    <div style={{ borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11px/1.5 var(--font-mono)', color: 'var(--text-secondary)' }}>
      {parts.join(' · ')}
    </div>
  );
}
