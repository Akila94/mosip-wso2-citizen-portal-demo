import React from 'react';
import { Card } from '../../design-system/components/core/Card';
import { Badge } from '../../design-system/components/core/Badge';
import { EffortContractLine } from './EffortContractLine';
import type { ServiceItem } from '../../services/types';

const STATE_TONE: Record<ServiceItem['state'], 'brand' | 'neutral' | 'success' | 'warning'> = {
  LIVE: 'brand',
  STUB: 'neutral',
  READY: 'success',
  STEP_UP: 'warning',
};
const STATE_LABEL: Record<ServiceItem['state'], string> = {
  LIVE: 'LIVE',
  STUB: 'STUB',
  READY: 'READY',
  STEP_UP: 'STEP-UP',
};

export function ServiceCard({ service, signedIn, onClick }: { service: ServiceItem; signedIn: boolean; onClick?: () => void }) {
  return (
    <Card interactive={!!onClick} onClick={onClick}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-2)' }}>
          <h3 style={{ margin: 0, font: '600 14px/1.3 var(--font-sans)' }}>{service.title}</h3>
          <Badge tone={STATE_TONE[service.state]}>{STATE_LABEL[service.state]}</Badge>
        </div>
        <p style={{ margin: 0, font: '400 12px/1.45 var(--font-sans)', color: 'var(--text-secondary)' }}>{service.description}</p>
        <EffortContractLine effort={service.effort} signedIn={signedIn} />
      </div>
    </Card>
  );
}
