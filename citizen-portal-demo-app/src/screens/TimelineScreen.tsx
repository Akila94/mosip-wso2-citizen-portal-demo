import React, { useMemo, useState } from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { TimelineTable } from '../components/timeline/TimelineTable';
import { SubmittedApplicationsPanel } from '../components/timeline/SubmittedApplicationsPanel';
import { AsyncState } from '../components/common/AsyncState';
import { useTimelineData } from '../hooks/usePortalData';
import type { TimelineItem } from '../services/types';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

type Filter = 'All' | TimelineItem['chip'];
const FILTERS: Filter[] = ['All', 'Action needed', 'Waiting on government', 'Appointment'];

export function TimelineScreen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const { data, loading, error, reload } = useTimelineData();
  const [filter, setFilter] = useState<Filter>('All');

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    (data ?? []).forEach((i) => {
      c[i.chip] = (c[i.chip] ?? 0) + 1;
    });
    return c;
  }, [data]);

  const filtered = useMemo(() => (data ?? []).filter((i) => filter === 'All' || i.chip === filter), [data, filter]);

  return (
    <>
      <AppHeader screen="timeline" onNavigate={onNavigate} breadcrumb={['My Timeline']} />
      <main className={styles.page}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)', flexWrap: 'wrap' }}>
          <h1 style={{ margin: 0, font: '700 26px var(--font-display)' }}>My timeline</h1>
          <p style={{ margin: 0, font: '400 14px var(--font-sans)', color: 'var(--text-secondary)' }}>What you owe government, and what government owes you.</p>
        </div>
        <div role="tablist" aria-label="Filter timeline" style={{ display: 'flex', gap: 'var(--space-2)', flexWrap: 'wrap' }}>
          {FILTERS.map((f) => (
            <button
              key={f}
              type="button"
              role="tab"
              aria-selected={filter === f}
              onClick={() => setFilter(f)}
              style={{
                padding: '8px 14px',
                borderRadius: 'var(--radius-pill)',
                font: '500 12px var(--font-mono)',
                cursor: 'pointer',
                background: filter === f ? 'var(--gray-950)' : 'transparent',
                color: filter === f ? 'var(--gray-0)' : 'var(--text-primary)',
                border: filter === f ? '1.5px solid var(--gray-950)' : '1.5px solid var(--border-strong)',
              }}
            >
              {f}
              {f !== 'All' && counts[f] ? ` (${counts[f]})` : ''}
            </button>
          ))}
        </div>
        <AsyncState loading={loading} error={error} onRetry={reload} isEmpty={filtered.length === 0} emptyMessage="No items match this filter.">
          <TimelineTable items={filtered} />
        </AsyncState>
        <SubmittedApplicationsPanel />
      </main>
      <AppFooter />
    </>
  );
}
