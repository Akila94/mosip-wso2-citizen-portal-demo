import React from 'react';
import { useSessionInspector } from '../../hooks/useSessionInspector';
import { AsyncState } from '../common/AsyncState';

/** Demo tool (frame 21): toggled from the footer, collapsed by default,
 * meant to be hidden for non-technical audiences. Shows the same session
 * id/IdP behind two different micro apps with different released claims —
 * scope, not session, decides what each app sees. */
export function SessionInspectorPanel({ clientId, onCollapse }: { clientId: string; onCollapse: () => void }) {
  const { data, loading, error, reload } = useSessionInspector(clientId, true);

  return (
    <div style={{ borderTop: '2px solid var(--border-strong)', background: 'var(--surface-card)', padding: 'var(--space-6)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-3)' }}>
          <h2 style={{ margin: 0, font: '600 15px var(--font-sans)' }}>Session inspector</h2>
          <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>demo tool · toggled from the footer · hidden for non-technical rooms</span>
        </div>
        <button type="button" onClick={onCollapse} style={{ padding: '6px 10px', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', background: 'var(--surface-card)', font: '500 12px var(--font-mono)', cursor: 'pointer' }}>
          ▾ collapse
        </button>
      </div>
      <AsyncState loading={loading} error={error} onRetry={reload}>
        {data && (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Session</span>
                {data.sessionRows.map((r) => (
                  <div key={r.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-3)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11.5px var(--font-mono)' }}>
                    <span style={{ color: 'var(--text-secondary)' }}>{r.label}</span>
                    <span style={{ fontWeight: 500, textAlign: 'right' }}>{r.value}</span>
                  </div>
                ))}
              </div>
              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>ID token claims — as released to this app</span>
                <pre style={{ margin: 0, border: '1.5px dashed var(--border-strong)', background: 'var(--surface-sunken)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', font: '400 11px/1.75 var(--font-mono)', color: 'var(--text-primary)', whiteSpace: 'pre-wrap', overflowX: 'auto' }}>
                  {JSON.stringify(data.claims, null, 2)}
                </pre>
                <span style={{ font: '400 11px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{data.comparisonNote}</span>
              </div>
            </div>
            <p style={{ margin: 0, font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
              Open this on the Vehicle Revenue Licence screen and compare with the same panel on the Driving Licence application: identical session id and IdP, different claims.
            </p>
          </>
        )}
      </AsyncState>
    </div>
  );
}
