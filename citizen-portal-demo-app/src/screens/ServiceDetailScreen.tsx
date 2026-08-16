import React from 'react';
import { AppHeader } from '../components/layout/AppHeader';
import { AppFooter } from '../components/layout/AppFooter';
import { AsyncState } from '../components/common/AsyncState';
import { useServiceDetail } from '../hooks/useIdentityData';
import { useAuth } from '../context/AuthContext';
import { Card } from '../design-system/components/core/Card';
import { Button } from '../design-system/components/core/Button';
import type { Screen } from '../App';
import styles from '../styles/layout.module.css';

export interface ServiceDetailScreenProps {
  serviceId: string;
  onNavigate: (screen: Screen) => void;
  onStartApplication: (serviceId: string) => void;
  onHeaderSignIn: () => void;
}

export function ServiceDetailScreen({ serviceId, onNavigate, onStartApplication, onHeaderSignIn }: ServiceDetailScreenProps) {
  const { data: service, loading, error, reload } = useServiceDetail(serviceId);
  const { isAuthenticated } = useAuth();

  return (
    <>
      <AppHeader screen="serviceDetail" onNavigate={onNavigate} breadcrumb={service ? [service.category, service.title] : ['Service']} onSignIn={onHeaderSignIn} />
      <main className={styles.page}>
        <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading service details…">
          {service && (
            <div style={{ display: 'flex', gap: 'var(--space-8)', alignItems: 'flex-start', flexWrap: 'wrap' }}>
              <div style={{ flex: '2 1 560px', display: 'flex', flexDirection: 'column', gap: 'var(--space-8)' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                  <span style={{ font: '600 11px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
                    {service.category} · {service.department}
                  </span>
                  <h1 style={{ margin: 0, font: '700 30px var(--font-display)' }}>{service.title}</h1>
                  <p style={{ margin: 0, maxWidth: 640, font: '400 14px/1.6 var(--font-sans)', color: 'var(--text-secondary)' }}>{service.summary}</p>
                </div>

                {service.sections.map((section) => (
                  <section key={section.id} aria-labelledby={`sec-${section.id}`} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)', borderBottom: '1.5px solid var(--border-default)', paddingBottom: 'var(--space-2)' }}>
                      <h2 id={`sec-${section.id}`} style={{ margin: 0, font: '600 16px var(--font-sans)' }}>{section.heading}</h2>
                      {section.note && <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-muted)' }}>{section.note}</span>}
                    </div>
                    <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                      {section.items.map((item, i) => (
                        <li key={i} style={{ display: 'flex', gap: 'var(--space-2)', font: '400 13px/1.6 var(--font-sans)', color: 'var(--text-primary)' }}>
                          <span aria-hidden="true" style={{ color: 'var(--text-muted)' }}>—</span>
                          {item}
                        </li>
                      ))}
                    </ul>
                  </section>
                ))}

                <section aria-labelledby="channels-heading" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                  <h2 id="channels-heading" style={{ margin: 0, font: '600 16px var(--font-sans)', borderBottom: '1.5px solid var(--border-default)', paddingBottom: 'var(--space-2)' }}>
                    Other ways to get this service
                  </h2>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 'var(--space-3)' }}>
                    {service.channels.map((c) => (
                      <Card key={c.id}>
                        <span style={{ display: 'block', font: '600 13px var(--font-sans)', marginBottom: 4 }}>{c.title}</span>
                        <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{c.description}</span>
                      </Card>
                    ))}
                  </div>
                </section>
              </div>

              <aside style={{ flex: '1 1 320px', maxWidth: 380, display: 'flex', flexDirection: 'column', gap: 'var(--space-4)', position: 'sticky', top: 'var(--space-4)' }}>
                <Card>
                  <span style={{ display: 'block', font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 'var(--space-3)' }}>Cost &amp; time</span>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                    <Row label="Fee" value={service.cost.fee} big />
                    <Row label="Processing time" value={service.cost.processingTime} />
                    <Row label="Form length" value={service.cost.formLength} />
                    <Row label="Payment" value={service.cost.payment} />
                  </div>
                </Card>

                <Card>
                  <span style={{ display: 'block', font: '600 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 'var(--space-3)' }}>What you'll sign in with</span>
                  <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                    {service.signInWith.map((s, i) => (
                      <li key={i} style={{ display: 'flex', gap: 'var(--space-2)', borderTop: i > 0 ? '1px dashed var(--border-default)' : 'none', paddingTop: i > 0 ? 'var(--space-2)' : 0, font: '400 12px/1.5 var(--font-sans)' }}>
                        <span aria-hidden="true" style={{ color: 'var(--text-secondary)' }}>›</span>
                        {s}
                      </li>
                    ))}
                  </ul>
                </Card>

                <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                  <Button fullWidth onClick={() => onStartApplication(service.id)}>{isAuthenticated ? 'Start application' : 'Sign in to start application'}</Button>
                  <Button variant="secondary" fullWidth onClick={() => {}}>Save to check later</Button>
                </div>
              </aside>
            </div>
          )}
        </AsyncState>
      </main>
      <AppFooter />
    </>
  );
}

function Row({ label, value, big }: { label: string; value: string; big?: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)' }}>
      <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ font: `600 ${big ? '20px' : '14px'} var(--font-sans)` }}>{value}</span>
    </div>
  );
}
