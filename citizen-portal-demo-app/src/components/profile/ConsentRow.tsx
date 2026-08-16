import React from 'react';
import { Button } from '../../design-system/components/core/Button';
import type { ConsentGrant } from '../../services/types';
import styles from '../../styles/layout.module.css';

export function ConsentRow({ consent, onRevoke }: { consent: ConsentGrant; onRevoke: (id: string) => void }) {
  return (
    <div className={styles.consentRow}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <span style={{ font: '600 13.5px var(--font-sans)' }}>{consent.appName}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{consent.agency}</span>
      </div>
      <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>scopes: {consent.scopes.join(' · ')}</span>
      <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-muted)' }}>granted {consent.grantedDate}</span>
      <Button variant="secondary" size="sm" onClick={() => onRevoke(consent.id)}>
        Revoke
      </Button>
    </div>
  );
}
