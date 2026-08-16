import React from 'react';
import { Card } from '../../design-system/components/core/Card';
import { Placeholder } from '../common/Placeholder';
import type { LifeEvent } from '../../services/types';

export function LifeEventCard({ event, onClick }: { event: LifeEvent; onClick?: () => void }) {
  return (
    <Card interactive onClick={onClick} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
      <Placeholder label="icon" width={30} height={30} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <span style={{ font: '600 13.5px var(--font-sans)' }}>{event.title}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{event.serviceCount}</span>
      </div>
    </Card>
  );
}
