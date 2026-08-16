import React from 'react';
import { Placeholder } from '../common/Placeholder';
import { Badge } from '../../design-system/components/core/Badge';
import { useAuth } from '../../context/AuthContext';

export interface MicroAppHeaderProps {
  onSaveExit: () => void;
  appName?: string;
  tagline?: string;
}

/** Generalized for reuse across every micro app (Document C's Driving
 * Licence Service, Document D's Vehicle Revenue Licence) — the wireframe
 * repeats this exact header per micro app with only the name/tagline
 * changing, so it's one component with props rather than a copy per app. */
export function MicroAppHeader({ onSaveExit, appName = 'Driving Licence Service', tagline = 'micro app · Dept. of Motor Traffic · within MaroliaGov' }: MicroAppHeaderProps) {
  const { user, assuranceLevel } = useAuth();
  return (
    <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 'var(--space-4)', padding: 'var(--space-3) var(--space-8)', borderBottom: '1px solid var(--border-default)', flexWrap: 'wrap' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        <Placeholder label="app" width={32} height={32} />
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <span style={{ font: '600 13.5px var(--font-sans)' }}>{appName}</span>
          <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{tagline}</span>
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
        {assuranceLevel === 'substantial' && <Badge tone="success" dot>Verified account</Badge>}
        <span style={{ font: '400 12px var(--font-sans)' }}>{user?.displayName ?? 'J. Doe'}</span>
        <button type="button" onClick={onSaveExit} style={{ background: 'none', border: 'none', font: '500 11px var(--font-mono)', color: 'var(--text-secondary)', cursor: 'pointer', textDecoration: 'underline' }}>
          Save &amp; exit
        </button>
      </div>
    </header>
  );
}
