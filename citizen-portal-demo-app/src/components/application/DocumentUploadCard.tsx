import React, { useRef } from 'react';
import { Badge } from '../../design-system/components/core/Badge';
import { Placeholder } from '../common/Placeholder';
import type { UploadedDoc } from '../../hooks/useLicenceApplicationWizard';

export interface DocumentUploadCardProps {
  title: string;
  spec: string;
  file: UploadedDoc | null;
  onFileSelected: (doc: UploadedDoc | null) => void;
  required?: boolean;
}

function formatSize(bytes: number) {
  return bytes > 1024 * 1024 ? `${(bytes / (1024 * 1024)).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export function DocumentUploadCard({ title, spec, file, onFileSelected, required }: DocumentUploadCardProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (f) onFileSelected({ name: f.name, sizeLabel: formatSize(f.size) });
  }

  return (
    <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <span style={{ font: '600 14px var(--font-sans)' }}>{title}</span>
        <Badge tone={file ? 'brand' : 'neutral'}>{file ? 'UPLOADED' : required ? 'REQUIRED' : 'OPTIONAL'}</Badge>
      </div>
      <div style={{ font: '400 11.5px/1.6 var(--font-mono)', color: 'var(--text-secondary)' }}>{spec}</div>
      {file ? (
        <div style={{ display: 'flex', gap: 'var(--space-3)', alignItems: 'center', border: '1px solid var(--border-default)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', background: 'var(--surface-sunken)' }}>
          <Placeholder label="file" width={40} height={50} />
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <span style={{ font: '600 12.5px var(--font-sans)' }}>{file.name}</span>
            <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>{file.sizeLabel}</span>
          </div>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 'var(--space-3)' }}>
            <button type="button" onClick={() => inputRef.current?.click()} style={{ background: 'none', border: 'none', padding: 0, font: '500 11.5px var(--font-sans)', textDecoration: 'underline', cursor: 'pointer' }}>Replace</button>
            <button type="button" onClick={() => onFileSelected(null)} style={{ background: 'none', border: 'none', padding: 0, font: '500 11.5px var(--font-sans)', textDecoration: 'underline', cursor: 'pointer' }}>Remove</button>
          </div>
        </div>
      ) : (
        <div style={{ border: '1.5px dashed var(--border-strong)', background: 'var(--surface-sunken)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-6)', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 'var(--space-2)' }}>
          <span style={{ font: '600 12.5px var(--font-sans)' }}>Drop a file here, or</span>
          <button type="button" onClick={() => inputRef.current?.click()} style={{ padding: '11px 18px', border: '1.5px solid var(--border-strong)', background: 'var(--surface-card)', borderRadius: 'var(--radius-sm)', font: '600 12.5px var(--font-sans)', cursor: 'pointer' }}>
            Choose a file
          </button>
          <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-muted)' }}>or take a photo with your phone camera</span>
        </div>
      )}
      <input ref={inputRef} type="file" accept=".pdf,.jpg,.jpeg,.png" onChange={handleChange} style={{ display: 'none' }} aria-label={`Upload ${title}`} />
    </div>
  );
}
