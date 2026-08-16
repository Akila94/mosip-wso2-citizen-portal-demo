import React from 'react';
import { ChevronRight } from 'lucide-react';
import { Card } from '../../design-system/components/core/Card';
import { Placeholder } from '../common/Placeholder';
import type { IdentityProvider } from '../../services/types';

export function FederatedProviderRow({ provider, onClick }: { provider: IdentityProvider; onClick: () => void }) {
  return (
    <Card interactive onClick={onClick} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)' }}>
      <Placeholder label="idp" width={30} height={30} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2, flex: 1 }}>
        <span style={{ font: '600 13.5px var(--font-sans)' }}>{provider.name}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{provider.description}</span>
      </div>
      <ChevronRight size={18} color="var(--text-muted)" aria-hidden="true" />
    </Card>
  );
}
