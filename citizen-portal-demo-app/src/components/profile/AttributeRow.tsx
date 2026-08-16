import React from 'react';
import { Lock } from 'lucide-react';
import type { AttributeRecord } from '../../services/types';
import styles from '../../styles/layout.module.css';

export function AttributeRow({ attribute, onEdit }: { attribute: AttributeRecord; onEdit?: (attribute: AttributeRecord) => void }) {
  return (
    <div className={styles.attributesRow}>
      <span style={{ font: '400 13px var(--font-sans)', color: 'var(--text-secondary)' }}>{attribute.label}</span>
      <span style={{ font: '600 13.5px var(--font-sans)' }}>{attribute.value}</span>
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, alignSelf: 'start', font: '500 11px var(--font-mono)', padding: '5px 9px', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-pill)' }}>
        {attribute.source === 'verified' && <Lock size={11} aria-hidden="true" />}
        {attribute.sourceLabel}
      </span>
      {attribute.editable ? (
        <button type="button" onClick={() => onEdit?.(attribute)} style={{ background: 'none', border: 'none', padding: 0, font: '600 12px var(--font-sans)', color: 'var(--text-link)', textDecoration: 'underline', cursor: 'pointer', justifySelf: 'start' }}>
          Edit
        </button>
      ) : (
        <span style={{ font: '600 12px var(--font-sans)', color: 'var(--text-muted)' }}>Read-only</span>
      )}
    </div>
  );
}
