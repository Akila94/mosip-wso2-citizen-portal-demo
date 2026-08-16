import React, { useState } from 'react';
import { Placeholder } from '../components/common/Placeholder';
import { SelectableTile } from '../components/application/SelectableTile';
import { AsyncState } from '../components/common/AsyncState';
import { Button } from '../design-system/components/core/Button';
import { Badge } from '../design-system/components/core/Badge';
import { useApplicationConfig } from '../hooks/useApplicationData';
import type { Screen } from '../App';

/**
 * Frame 23 — an alternative to frame 13 (Document C's Step 1), drawn for
 * comparison only. Per the wireframe: "compare against frame 13. Canonical
 * demo flow stays as drawn in documents B and C; this exists so the choice
 * is a decision you made, not one you defaulted into." Reachable only via
 * the demo nav, never wired into the canonical A→B→C flow.
 */
export function DeferredAuthStep1Screen({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  const { data: config, loading, error, reload } = useApplicationConfig();
  const [appTypeId, setAppTypeId] = useState('new');
  const [selectedClassIds, setSelectedClassIds] = useState<string[]>(['b']);
  const [dob, setDob] = useState('04 Mar 1996');
  const [saved, setSaved] = useState(false);

  function toggleClass(id: string) {
    setSelectedClassIds((prev) => (prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]));
  }

  return (
    <>
      <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 'var(--space-3) var(--space-8)', borderBottom: '1px solid var(--border-default)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
          <Placeholder label="app" width={32} height={32} />
          <span style={{ font: '600 13.5px var(--font-sans)' }}>Driving Licence Service</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
          <Badge tone="neutral">NOT SIGNED IN · ANSWERS SAVED LOCALLY</Badge>
          <Button onClick={() => onNavigate('identityLogin')}>Sign in</Button>
        </div>
      </header>

      <nav aria-label="Application progress" style={{ display: 'flex', borderBottom: '1px solid var(--border-default)' }}>
        {[
          { n: 1, label: 'Eligibility & class', state: 'current · anonymous' },
          { n: 2, label: 'Applicant details', state: 'sign-in needed here' },
          { n: 3, label: 'Medical & documents', state: 'sign-in needed' },
          { n: 4, label: 'Appointment, review & pay', state: 'sign-in needed' },
        ].map((s) => (
          <div key={s.n} style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 'var(--space-3)', padding: 'var(--space-3) var(--space-4)', borderRight: '1px dashed var(--border-default)', background: s.n === 1 ? 'var(--surface-sunken)' : 'transparent' }}>
            <span style={{ flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', width: 26, height: 26, borderRadius: '50%', border: `1.5px ${s.n === 1 ? 'solid' : 'dashed'} var(--border-strong)`, background: s.n === 1 ? 'var(--gray-950)' : 'var(--surface-card)', color: s.n === 1 ? 'var(--gray-0)' : 'var(--text-primary)', font: '600 12px var(--font-mono)' }}>
              {s.n}
            </span>
            <span style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <span style={{ font: '600 12.5px var(--font-sans)' }}>{s.label}</span>
              <span style={{ font: '400 10.5px var(--font-mono)', color: 'var(--text-secondary)' }}>{s.state}</span>
            </span>
          </div>
        ))}
      </nav>

      <AsyncState loading={loading} error={error} onRetry={reload}>
        {config && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div style={{ flex: 1, padding: 'var(--space-8)', display: 'flex', flexDirection: 'column', gap: 'var(--space-6)', maxWidth: 900 }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Eligibility &amp; licence class</h1>
                <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>You can answer these without an account. We'll ask who you are only when it matters.</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <span style={{ font: '600 13.5px var(--font-sans)' }}>What are you applying for?</span>
                <div style={{ display: 'flex', gap: 'var(--space-3)' }}>
                  {config.appTypes.map((t) => (
                    <SelectableTile key={t.id} type="radio" name="app-type" label={t.label} description={t.description} selected={appTypeId === t.id} onSelect={() => setAppTypeId(t.id)} />
                  ))}
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <span style={{ font: '600 13.5px var(--font-sans)' }}>Licence classes</span>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 'var(--space-3)' }}>
                  {config.licenceClasses.slice(0, 4).map((c) => (
                    <SelectableTile key={c.id} type="checkbox" label={c.label} description={c.description} selected={selectedClassIds.includes(c.id)} onSelect={() => toggleClass(c.id)} />
                  ))}
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-5)' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <span style={{ font: '600 13px var(--font-sans)' }}>Your date of birth</span>
                  <input value={dob} onChange={(e) => setDob(e.target.value)} style={{ border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', font: '400 13px var(--font-mono)', color: 'var(--text-primary)', background: 'var(--surface-card)' }} />
                  <span style={{ font: '400 11px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>Typed, not verified — we check it against your identity record when you sign in. This is the cost of deferring.</span>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <span style={{ font: '600 13px var(--font-sans)' }}>Learner permit number</span>
                  <div style={{ border: '1px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', font: '400 13px var(--font-mono)' }}>{config.permitNumber}</div>
                  <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>Format LP-YYYY-NNNNN.</span>
                </div>
              </div>

              <div style={{ border: '2px solid var(--border-strong)', borderRadius: 'var(--radius-md)', background: 'var(--surface-sunken)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }} aria-live="polite">
                <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase' }}>Eligibility result — computed from what you typed</span>
                <span style={{ font: '600 15px var(--font-sans)' }}>Looks like you're eligible for class B</span>
                <span style={{ font: '400 12px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>Provisional. Confirmed once your identity record is checked at sign-in.</span>
              </div>

              <div style={{ border: '2.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-6)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
                <h2 style={{ margin: 0, font: '600 16px var(--font-sans)' }}>To continue, we now need to know who you are</h2>
                <p style={{ margin: 0, font: '400 12.5px/1.6 var(--font-sans)', color: 'var(--text-secondary)' }}>
                  Step 2 asks for your address and photograph. Rather than making you type them, sign in and we'll take them from your verified identity record — and keep the answers you have already given.
                </p>
                <div style={{ display: 'flex', gap: 'var(--space-3)', alignItems: 'center', flexWrap: 'wrap' }}>
                  <Button onClick={() => onNavigate('identityLogin')}>Sign in and continue</Button>
                  <Button variant="secondary" onClick={() => setSaved(true)}>Save my answers for later</Button>
                  <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)', flex: 1 }}>
                    {saved ? 'Saved in this browser for 24 hours and will be carried into the signed-in application.' : 'Your answers so far are held in this browser for 24 hours and carried into the signed-in application.'}
                  </span>
                </div>
              </div>
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
              <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Identity — not yet established</span>
              <div style={{ border: '1.5px dashed var(--border-strong)', background: 'var(--surface-card)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)', alignItems: 'center' }}>
                <Placeholder label="no photo" width={62} height={74} />
                <span style={{ font: '400 11.5px/1.55 var(--font-sans)', color: 'var(--text-secondary)', textAlign: 'center' }}>This panel fills with your verified details the moment you sign in. Everything in it is read-only.</span>
                <Button variant="secondary" onClick={() => onNavigate('identityLogin')}>Sign in now</Button>
              </div>
              <div style={{ border: '1px dotted var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '600 12px var(--font-sans)' }}>Trade-off, for your review</span>
                <span style={{ font: '400 11.5px/1.55 var(--font-sans)', color: 'var(--text-primary)' }}>
                  The login prompt feels earned rather than imposed, and drop-off before it is lower. The cost: some answers are self-asserted and must be re-checked, and the eligibility verdict is provisional until sign-in.
                </span>
              </div>
            </div>
          </div>
        )}
      </AsyncState>
    </>
  );
}
