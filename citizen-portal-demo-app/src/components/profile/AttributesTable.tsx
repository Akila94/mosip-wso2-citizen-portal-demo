import React from 'react';
import { AttributeRow } from './AttributeRow';
import type { AttributeRecord } from '../../services/types';
import styles from '../../styles/layout.module.css';

export function AttributesTable({ attributes, onEdit }: { attributes: AttributeRecord[]; onEdit?: (attribute: AttributeRecord) => void }) {
  return (
    <div className={styles.attributesTable}>
      <div className={styles.attributesHeadRow}>
        <span>Attribute</span>
        <span>Value</span>
        <span>Source</span>
        <span />
      </div>
      {attributes.map((a) => (
        <AttributeRow key={a.id} attribute={a} onEdit={onEdit} />
      ))}
    </div>
  );
}
