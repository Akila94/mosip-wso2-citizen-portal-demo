import React from 'react';
import type { VehicleRecord } from '../../services/types';
import { Button } from '../../design-system/components/core/Button';
import { Badge } from '../../design-system/components/core/Badge';

export function VehicleCard({ vehicle, onRenew, renewing, renewed }: { vehicle: VehicleRecord; onRenew: () => void; renewing: boolean; renewed: boolean }) {
  return (
    <div style={{ border: '2px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)', background: 'var(--surface-sunken)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={{ font: '600 16px var(--font-mono)' }}>{vehicle.plate}</span>
          <span style={{ font: '400 12px var(--font-sans)', color: 'var(--text-secondary)' }}>{vehicle.description}</span>
        </div>
        <Badge tone={renewed ? 'success' : 'neutral'}>{renewed ? 'RENEWED' : vehicle.dueDate}</Badge>
      </div>
      {vehicle.registryChecks.map((c) => (
        <div key={c.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-3)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 12px var(--font-sans)' }}>
          <span style={{ color: 'var(--text-secondary)' }}>{c.label}</span>
          <span style={{ fontWeight: 600, textAlign: 'right' }}>{c.status}</span>
        </div>
      ))}
      <Button fullWidth onClick={onRenew} disabled={renewing || renewed}>
        {renewed ? 'Renewed ✓' : renewing ? 'Processing…' : `Renew this licence — ${vehicle.fee}`}
      </Button>
    </div>
  );
}
