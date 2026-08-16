import React from 'react';
import { Alert } from '../../design-system/components/feedback/Alert';
import { Button } from '../../design-system/components/core/Button';

export interface AsyncStateProps {
  loading: boolean;
  error: string | null;
  isEmpty?: boolean;
  emptyMessage?: string;
  onRetry?: () => void;
  loadingLabel?: string;
  children: React.ReactNode;
}

/** Shared loading / error / empty presentation for every service-backed section. */
export function AsyncState({ loading, error, isEmpty, emptyMessage = 'Nothing to show yet.', onRetry, loadingLabel = 'Loading…', children }: AsyncStateProps) {
  if (loading) {
    return (
      <div role="status" aria-live="polite" style={{ padding: 'var(--space-8)', textAlign: 'center', color: 'var(--text-secondary)', fontFamily: 'var(--font-sans)', fontSize: 14 }}>
        {loadingLabel}
      </div>
    );
  }
  if (error) {
    return (
      <Alert tone="danger" title="Couldn't load this section">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          <span>{error}</span>
          {onRetry && (
            <Button variant="secondary" size="sm" onClick={onRetry} style={{ alignSelf: 'flex-start' }}>
              Try again
            </Button>
          )}
        </div>
      </Alert>
    );
  }
  if (isEmpty) {
    return (
      <div style={{ padding: 'var(--space-8)', textAlign: 'center', color: 'var(--text-secondary)', fontFamily: 'var(--font-sans)', fontSize: 14 }}>
        {emptyMessage}
      </div>
    );
  }
  return <>{children}</>;
}
