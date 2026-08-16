import React from 'react';
import { Card } from '../../design-system/components/core/Card';
import type { DepartmentRecord } from '../../services/types';

export function DepartmentRecordCard({ record }: { record: DepartmentRecord }) {
  return (
    <Card>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 'var(--space-3)' }}>
        <h3 style={{ margin: 0, font: '600 14px var(--font-sans)' }}>{record.department}</h3>
        <span style={{ font: '400 10.5px var(--font-mono)', color: 'var(--text-muted)' }}>read-only</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
        {record.rows.map((row) => (
          <div key={row.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-3)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 12px var(--font-sans)' }}>
            <span style={{ color: 'var(--text-secondary)' }}>{row.label}</span>
            <span style={{ fontWeight: 600 }}>{row.value}</span>
          </div>
        ))}
      </div>
    </Card>
  );
}
