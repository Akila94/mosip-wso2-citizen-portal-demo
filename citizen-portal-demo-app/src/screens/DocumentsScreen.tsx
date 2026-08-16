import React from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { WalletDocumentCard } from '../components/documents/WalletDocumentCard';
import { AsyncState } from '../components/common/AsyncState';
import { useDocumentsData } from '../hooks/usePortalData';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

export function DocumentsScreen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const { data, loading, error, reload } = useDocumentsData();

  return (
    <>
      <AppHeader screen="documents" onNavigate={onNavigate} breadcrumb={['My documents']} />
      <main className={styles.page}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)', flexWrap: 'wrap' }}>
          <h1 style={{ margin: 0, font: '700 26px var(--font-display)' }}>My documents</h1>
          <p style={{ margin: 0, font: '400 14px var(--font-sans)', color: 'var(--text-secondary)' }}>Credentials issued to you. Shareable, verifiable, revocable.</p>
        </div>
        <AsyncState loading={loading} error={error} onRetry={reload} isEmpty={data?.length === 0}>
          <ul className={styles.walletGrid} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
            {(data ?? []).map((doc) => (
              <li key={doc.id}>
                <WalletDocumentCard doc={doc} />
              </li>
            ))}
          </ul>
        </AsyncState>
      </main>
      <AppFooter />
    </>
  );
}
