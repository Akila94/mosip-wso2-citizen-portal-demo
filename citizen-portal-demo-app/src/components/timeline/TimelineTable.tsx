import React from 'react';
import { TimelineRow } from './TimelineRow';
import type { TimelineItem } from '../../services/types';
import styles from '../../styles/layout.module.css';

export function TimelineTable({ items, onAction }: { items: TimelineItem[]; onAction?: (item: TimelineItem) => void }) {
  return (
    <div className={styles.timelineTable}>
      <div className={styles.timelineHeadRow}>
        <span>When</span>
        <span>What</span>
        <span>Status</span>
        <span>Action</span>
      </div>
      {items.map((item) => (
        <TimelineRow key={item.id} item={item} onAction={onAction} />
      ))}
    </div>
  );
}
