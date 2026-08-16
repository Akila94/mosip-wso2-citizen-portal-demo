import React from 'react';
import { Badge } from '../../design-system/components/core/Badge';
import { Button } from '../../design-system/components/core/Button';
import type { TimelineItem } from '../../services/types';
import styles from '../../styles/layout.module.css';

const CHIP_TONE: Record<TimelineItem['chip'], 'warning' | 'info' | 'neutral'> = {
  'Action needed': 'warning',
  'Payment due': 'warning',
  Appointment: 'info',
  'Waiting on government': 'neutral',
  'For information': 'neutral',
};

export function TimelineRow({ item, onAction }: { item: TimelineItem; onAction?: (item: TimelineItem) => void }) {
  return (
    <div className={styles.timelineRow}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <span style={{ font: '600 13px var(--font-sans)' }}>{item.date}</span>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{item.relative}</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <span style={{ font: '600 14px var(--font-sans)' }}>{item.title}</span>
        <span style={{ font: '400 12px var(--font-sans)', color: 'var(--text-secondary)' }}>{item.description}</span>
      </div>
      <Badge tone={CHIP_TONE[item.chip]}>{item.chip}</Badge>
      <Button size="sm" onClick={() => onAction?.(item)}>
        {item.actionLabel}
      </Button>
    </div>
  );
}
