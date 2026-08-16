import React, { useState } from 'react';
import { Card } from '../../design-system/components/core/Card';
import type { SubmittedApplication } from '../../services/types';

const APPLICATIONS: SubmittedApplication[] = [
  { id: '1', reference: 'DL-2026-004871', title: 'Driving licence — new, class B', status: 'Under review' },
  { id: '2', reference: 'VR-2025-117220', title: 'Vehicle revenue licence — CAB-4471', status: 'Completed' },
  { id: '3', reference: 'CR-2025-009102', title: 'Birth certificate copy', status: 'Completed' },
];

/** Collapsed by default — history, not obligations, so it doesn't compete with the dated rows above. */
export function SubmittedApplicationsPanel() {
  const [open, setOpen] = useState(false);
  return (
    <Card padded={false}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        style={{ width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 'var(--space-3)', padding: 'var(--space-4) var(--space-5)', background: 'var(--surface-sunken)', border: 'none', borderRadius: open ? 'var(--radius-xl) var(--radius-xl) 0 0' : 'var(--radius-xl)', cursor: 'pointer', textAlign: 'left' }}
      >
        <span style={{ font: '600 14px var(--font-sans)' }}>
          {open ? '▾' : '▸'} Submitted applications ({APPLICATIONS.length})
        </span>
        <span style={{ font: '400 12px var(--font-mono)', color: 'var(--text-secondary)' }}>reference numbers · dates · status</span>
      </button>
      {open && (
        <div style={{ padding: 'var(--space-4) var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
          {APPLICATIONS.map((app) => (
            <div key={app.id} style={{ display: 'grid', gridTemplateColumns: '180px 1fr 140px', gap: 'var(--space-4)', font: '400 12px var(--font-mono)', color: 'var(--text-primary)' }}>
              <span>{app.reference}</span>
              <span>{app.title}</span>
              <span>{app.status}</span>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
