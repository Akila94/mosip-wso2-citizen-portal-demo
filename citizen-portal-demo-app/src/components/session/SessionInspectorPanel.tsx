import React, { useEffect, useState } from 'react';
import { useSessionInspector } from '../../hooks/useSessionInspector';
import { AsyncState } from '../common/AsyncState';
import type { SessionInspectorData, SessionRow } from '../../services/types';
import type { AppKey } from '../../config/clients';

/** Formats a Unix timestamp as a local wall-clock time, or a dash if absent. */
function clockTime(unixSeconds: number): string {
  if (!unixSeconds) return '—';
  return new Date(unixSeconds * 1000).toLocaleTimeString();
}

/**
 * Time left until `expiresAt`, counted down live. Switches to seconds in the
 * last minute so the session visibly running out is watchable during a demo —
 * this used to be the static string '51 min'.
 */
function remainingLifetime(expiresAt: number, nowSeconds: number): string {
  if (!expiresAt) return 'unknown';
  const secondsLeft = expiresAt - nowSeconds;
  if (secondsLeft <= 0) return 'expired';
  if (secondsLeft < 60) return `${secondsLeft} sec`;
  return `${Math.floor(secondsLeft / 60)} min ${secondsLeft % 60} sec`;
}

function buildRows(data: SessionInspectorData, nowSeconds: number): SessionRow[] {
  const rows: SessionRow[] = [
    { label: 'subject', value: data.subject },
    { label: 'idp', value: data.idp },
    { label: 'amr', value: data.amr.length ? data.amr.join(', ') : '—' },
  ];

  // WSO2 IS emits `acr` only when a value was actually resolved, so an absent
  // one is normal and is stated rather than faked.
  rows.push({ label: 'acr', value: data.acr || 'not asserted' });

  rows.push(
    { label: 'assurance', value: data.assuranceLevel },
    { label: 'sid', value: data.sid },
    { label: 'session started', value: clockTime(data.authTime) },
    { label: 'remaining lifetime', value: remainingLifetime(data.expiresAt, nowSeconds) },
    {
      label: 'clients in session',
      value: data.clientsInSession.length
        ? `${data.clientsInSession.length} — ${data.clientsInSession.map((c) => c.appName).join(', ')}`
        : '1 — this app only',
    }
  );

  return rows;
}

/**
 * Describes what this client received, from the data itself rather than from
 * hand-written per-client copy — so it stays true if scopes change.
 */
function comparisonNote(data: SessionInspectorData): string {
  const count = Object.keys(data.releasedClaims).length;
  return `${data.clientLabel} received ${count} claim${count === 1 ? '' : 's'} under client ${data.clientId}. Open the same panel on the other micro app: identical subject and sid, different audience and a different claim set.`;
}

/**
 * Demo tool (frame 21): toggled from the footer, collapsed by default, meant
 * to be hidden for non-technical audiences. Shows the same IdP session behind
 * two different micro apps with different released claims — scope, not
 * session, decides what each app sees.
 *
 * Every value here comes from the BFF's own verified session state. No token
 * is present in this payload or anywhere in the browser.
 */
export function SessionInspectorPanel({ appKey, onCollapse }: { appKey: AppKey; onCollapse: () => void }) {
  const { data, loading, error, reload } = useSessionInspector(appKey, true);
  const [nowSeconds, setNowSeconds] = useState(() => Math.floor(Date.now() / 1000));

  // Drives the live "remaining lifetime" countdown. Only ticks while the panel
  // is mounted, which is only while it is expanded.
  useEffect(() => {
    const interval = window.setInterval(() => setNowSeconds(Math.floor(Date.now() / 1000)), 1000);
    return () => window.clearInterval(interval);
  }, []);

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
                <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>Session — {data.clientLabel}</span>
                {buildRows(data, nowSeconds).map((r) => (
                  <div key={r.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-3)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11.5px var(--font-mono)' }}>
                    <span style={{ color: 'var(--text-secondary)', flexShrink: 0 }}>{r.label}</span>
                    <span style={{ fontWeight: 500, textAlign: 'right', wordBreak: 'break-all' }}>{r.value}</span>
                  </div>
                ))}
              </div>
              <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
                <span style={{ font: '500 10.5px var(--font-mono)', letterSpacing: 'var(--ls-eyebrow)', textTransform: 'uppercase', color: 'var(--text-secondary)' }}>ID token claims — as released to this app</span>
                <pre style={{ margin: 0, border: '1.5px dashed var(--border-strong)', background: 'var(--surface-sunken)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', font: '400 11px/1.75 var(--font-mono)', color: 'var(--text-primary)', whiteSpace: 'pre-wrap', overflowX: 'auto', maxHeight: 320 }}>
                  {JSON.stringify(data.releasedClaims, null, 2)}
                </pre>
                <span style={{ font: '400 11px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>{comparisonNote(data)}</span>
              </div>
            </div>
            <p style={{ margin: 0, font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
              These are the real claims WSO2 Identity Server released to this client. No access token or ID token is present in the browser — the BFF holds both and returns only this projection.
            </p>
          </>
        )}
      </AsyncState>
    </div>
  );
}
