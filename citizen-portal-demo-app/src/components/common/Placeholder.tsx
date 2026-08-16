import React from 'react';

export interface PlaceholderProps {
  label: string;
  width?: number | string;
  height?: number | string;
  shape?: 'square' | 'circle';
  style?: React.CSSProperties;
}

/**
 * Labeled dashed-border stand-in for real artwork (agency crest, citizen
 * photograph, credential QR code) that has no design-system equivalent and
 * no real asset yet. Kept as a clearly-labeled box rather than an icon, per
 * product decision.
 */
export function Placeholder({ label, width = '100%', height = '100%', shape = 'square', style }: PlaceholderProps) {
  return (
    <div
      role="img"
      aria-label={label}
      style={{
        width,
        height,
        border: '1.5px dashed var(--border-strong)',
        background: 'var(--surface-sunken)',
        borderRadius: shape === 'circle' ? '50%' : 'var(--radius-sm)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--text-muted)',
        fontFamily: 'var(--font-mono)',
        fontSize: 10,
        textAlign: 'center',
        lineHeight: 1.3,
        flexShrink: 0,
        ...style,
      }}
    >
      {label}
    </div>
  );
}
