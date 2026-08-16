import React from 'react';
import { Button } from '../../design-system/components/core/Button';

export function ApplicationFooterNav({ backLabel, onBack, continueLabel, onContinue, autoSaveNote }: { backLabel: string; onBack: () => void; continueLabel: string; onContinue: () => void; autoSaveNote?: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1.5px solid var(--border-default)', paddingTop: 'var(--space-5)' }}>
      <Button variant="secondary" onClick={onBack}>{backLabel}</Button>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        {autoSaveNote && <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>saved automatically</span>}
        <Button onClick={onContinue}>{continueLabel}</Button>
      </div>
    </div>
  );
}
