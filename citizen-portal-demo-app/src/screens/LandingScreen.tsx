import React from 'react';
import { Search } from 'lucide-react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { LifeEventCard } from '../components/service/LifeEventCard';
import { ServiceCatalogue } from '../components/service/ServiceCatalogue';
import { TimelineTable } from '../components/timeline/TimelineTable';
import { AsyncState } from '../components/common/AsyncState';
import { useAuth } from '../context/AuthContext';
import { useLandingData } from '../hooks/usePortalData';
import { Input } from '../design-system/components/forms/Input';
import { Button } from '../design-system/components/core/Button';
import { Tag } from '../design-system/components/core/Tag';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

export function LandingScreen({ onNavigate, onSelectService, onHeaderSignIn }: { onNavigate: (screen: Screen) => void; onSelectService: (serviceId: string) => void; onHeaderSignIn: () => void }) {
  const { isAuthenticated } = useAuth();
  const { lifeEvents, categories, timelinePreview } = useLandingData();

  return (
    <>
      <AppHeader screen="landing" onNavigate={onNavigate} onSignIn={onHeaderSignIn} />
      <main className={styles.page}>
        {!isAuthenticated ? (
          <section style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-5)', textAlign: 'center', padding: 'var(--space-8) 0' }}>
            <h1 style={{ margin: 0, font: '700 34px/1.2 var(--font-display)', letterSpacing: 'var(--ls-heading)', maxWidth: 760 }}>
              MaroliaGov: Every government service, one account
            </h1>
            <p style={{ margin: 0, font: '400 15px var(--font-sans)', color: 'var(--text-secondary)', maxWidth: 560 }}>
              Find what you need, check what it costs and how long it takes — and sign in only when you're ready to apply.
            </p>
            <form role="search" style={{ display: 'flex', gap: 'var(--space-2)', width: '100%', maxWidth: 680, alignItems: 'flex-end' }} onSubmit={(e) => e.preventDefault()}>
              <Input
                label="Ask or search"
                placeholder='e.g. "renew my driving licence" or "what do I need to register a birth?"'
                iconLeft={<Search size={16} />}
                containerStyle={{ flex: 1, textAlign: 'left' }}
              />
              <Button type="submit" style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                Search
                <Tag style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.6)', color: '#fff', height: 20, padding: '0 6px', fontSize: 10, fontWeight: 600, letterSpacing: '0.04em' }}>AI</Tag>
              </Button>
            </form>
          </section>
        ) : (
          <section aria-labelledby="timeline-preview-heading" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
              <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)' }}>
                <h2 id="timeline-preview-heading" style={{ margin: 0, font: '600 18px var(--font-sans)' }}>Your timeline</h2>
                <span style={{ font: '400 12px var(--font-mono)', color: 'var(--text-secondary)' }}>next 60 days</span>
              </div>
              <button type="button" onClick={() => onNavigate('timeline')} style={{ background: 'none', border: 'none', font: '600 12.5px var(--font-sans)', color: 'var(--text-link)', textDecoration: 'underline', cursor: 'pointer' }}>
                Open My Timeline →
              </button>
            </div>
            <AsyncState loading={timelinePreview.loading} error={timelinePreview.error} onRetry={timelinePreview.reload} isEmpty={timelinePreview.data?.length === 0} emptyMessage="Nothing due in the next 60 days.">
              <TimelineTable items={timelinePreview.data ?? []} />
            </AsyncState>
          </section>
        )}

        <section aria-labelledby="life-events-heading">
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)', marginBottom: 'var(--space-4)' }}>
            <h2 id="life-events-heading" style={{ margin: 0, font: '600 18px var(--font-sans)' }}>Life events</h2>
            <span style={{ font: '400 12px var(--font-mono)', color: 'var(--text-secondary)' }}>start from a moment, not a ministry</span>
          </div>
          <AsyncState loading={lifeEvents.loading} error={lifeEvents.error} onRetry={lifeEvents.reload} isEmpty={lifeEvents.data?.length === 0}>
            <ul className={styles.lifeEventGrid} style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              {(lifeEvents.data ?? []).map((event) => (
                <li key={event.id}>
                  <LifeEventCard event={event} />
                </li>
              ))}
            </ul>
          </AsyncState>
        </section>

        <section aria-labelledby="catalogue-heading">
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)', marginBottom: 'var(--space-5)' }}>
            <h2 id="catalogue-heading" style={{ margin: 0, font: '600 18px var(--font-sans)' }}>Service catalogue</h2>
            <span style={{ font: '400 12px var(--font-mono)', color: 'var(--text-secondary)' }}>grouped by category · every card carries the effort contract</span>
          </div>
          <AsyncState loading={categories.loading} error={categories.error} onRetry={categories.reload} isEmpty={categories.data?.length === 0}>
            <ServiceCatalogue categories={categories.data ?? []} signedIn={isAuthenticated} onSelectService={onSelectService} />
          </AsyncState>
        </section>
      </main>
      <AppFooter compact={!isAuthenticated} />
    </>
  );
}
