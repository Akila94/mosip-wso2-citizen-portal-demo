import React from 'react';
import { Button } from '../../design-system/components/core/Button';
import type { ConsentGrant } from '../../services/types';
import styles from '../../styles/layout.module.css';

/**
 * Revoking a grant means calling WSO2 Identity Server's consent-management
 * API, which is a separate integration from this one — `gov-services-api`
 * serves consents read-only and exposes no revoke route. The control is
 * therefore disabled and says why, rather than being wired to a write path
 * that would appear to succeed and silently revert on the next reload.
 */
export function ConsentRow({ consent }: { consent: ConsentGrant }) {
  return (
    <div className={styles.consentRow}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <span style={{ font: '600 13.5px var(--font-sans)' }}>{consent.appName}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{consent.agency}</span>
      </div>
      <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>scopes: {consent.scopes.join(' · ')}</span>
      <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-muted)' }}>granted {consent.grantedDate}</span>
      <Button variant="secondary" size="sm" disabled title="Revoking a grant is managed in WSO2 Identity Server; not available from the portal yet.">
        Revoke
      </Button>
    </div>
  );
}
