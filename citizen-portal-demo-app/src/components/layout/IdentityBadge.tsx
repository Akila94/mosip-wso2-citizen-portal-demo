import React from 'react';
import { Badge } from '../../design-system/components/core/Badge';
import type { AssuranceLevel } from '../../services/types';

export function IdentityBadge({ assuranceLevel }: { assuranceLevel: AssuranceLevel }) {
  if (assuranceLevel === 'none') return null;
  const verified = assuranceLevel === 'substantial';
  return (
    <Badge tone={verified ? 'success' : 'neutral'} dot>
      {verified ? 'Verified account' : 'Basic account — verify your identity'}
    </Badge>
  );
}
