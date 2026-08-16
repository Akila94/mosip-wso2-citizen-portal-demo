import React from 'react';
import { Card } from '../../design-system/components/core/Card';
import { Badge } from '../../design-system/components/core/Badge';
import { Button } from '../../design-system/components/core/Button';
import { Placeholder } from '../common/Placeholder';
import type { WalletDocument } from '../../services/types';

const STATUS_TONE: Record<WalletDocument['status'], 'success' | 'neutral' | 'brand'> = {
  VALID: 'success',
  NOT_ISSUED: 'neutral',
  NEW: 'brand',
};
const STATUS_LABEL: Record<WalletDocument['status'], string> = {
  VALID: 'VALID',
  NOT_ISSUED: 'NOT ISSUED',
  NEW: 'NEW · ISSUED TODAY',
};

export function WalletDocumentCard({ doc, onPrimary, onSecondary }: { doc: WalletDocument; onPrimary?: () => void; onSecondary?: () => void }) {
  const notIssued = doc.status === 'NOT_ISSUED';
  const primaryVariant = notIssued || doc.status === 'NEW' ? 'primary' : 'secondary';
  return (
    <Card style={{ minHeight: 200, display: 'flex', flexDirection: 'column', ...(notIssued ? { borderStyle: 'dashed', background: 'var(--surface-sunken)' } : {}) }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 'var(--space-2)', marginBottom: 'var(--space-3)' }}>
        <h3 style={{ margin: 0, font: '600 15px var(--font-sans)' }}>{doc.title}</h3>
        <Badge tone={STATUS_TONE[doc.status]}>{STATUS_LABEL[doc.status]}</Badge>
      </div>
      <div style={{ display: 'flex', gap: 'var(--space-4)', marginBottom: 'var(--space-4)' }}>
        <Placeholder label="QR code" width={74} height={74} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6, font: '400 12px var(--font-sans)', color: 'var(--text-secondary)' }}>
          <span>Number · {doc.number}</span>
          <span>Issued · {doc.issuedDate}</span>
          <span>Expires · {doc.expiryDate}</span>
        </div>
      </div>
      <div style={{ marginTop: 'auto', display: 'flex', gap: 'var(--space-2)' }}>
        <Button variant={primaryVariant} fullWidth onClick={onPrimary}>
          {doc.primaryAction}
        </Button>
        <Button variant="ghost" fullWidth onClick={onSecondary}>
          {doc.secondaryAction}
        </Button>
      </div>
    </Card>
  );
}
