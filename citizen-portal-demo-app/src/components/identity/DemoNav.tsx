import React from 'react';
import { DEMO_ROUTES } from './demoRoutes';
import type { Screen } from '../../App';

/** Dev-only nav for frames with no organic entry point yet (step-up MFA and
 * the session-expired modal are triggered from Document C frames that don't
 * exist yet). Not part of the wireframe — flagged as prototype scaffolding
 * to remove once Document C wires these in for real. */
export function DemoNav({ onNavigate }: { onNavigate: (screen: Screen) => void }) {
  return (
    <div style={{ position: 'fixed', left: 'var(--space-3)', bottom: 'var(--space-3)', zIndex: 50, display: 'flex', flexDirection: 'column', gap: 4, background: 'var(--gray-950)', color: 'var(--gray-0)', borderRadius: 'var(--radius-md)', padding: 'var(--space-3)', boxShadow: 'var(--shadow-lg)', font: 'var(--font-mono)' }}>
      <span style={{ font: '600 9px var(--font-mono)', letterSpacing: '0.1em', textTransform: 'uppercase', opacity: 0.6 }}>Demo entry points</span>
      {DEMO_ROUTES.map((r) => (
        <button key={r.screen} type="button" onClick={() => onNavigate(r.screen)} style={{ background: 'none', border: 'none', color: 'var(--gray-0)', font: '500 11px var(--font-sans)', textAlign: 'left', padding: '2px 0', cursor: 'pointer', textDecoration: 'underline' }}>
          {r.label}
        </button>
      ))}
    </div>
  );
}
