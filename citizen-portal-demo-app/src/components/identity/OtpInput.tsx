import React, { useRef } from 'react';

export interface OtpInputProps {
  length?: number;
  value: string;
  onChange: (value: string) => void;
}

/** Deviation: no OTP/code-input primitive in the design system. Built from
 * DS color/spacing/radius tokens; exposed to assistive tech as one labelled
 * field via aria-label + a visually hidden combined value, with paste support. */
export function OtpInput({ length = 6, value, onChange }: OtpInputProps) {
  const refs = useRef<(HTMLInputElement | null)[]>([]);
  const digits = value.padEnd(length, ' ').split('').slice(0, length);

  function setDigit(index: number, char: string) {
    const next = digits.slice();
    next[index] = char;
    onChange(next.join('').replace(/\s/g, ''));
  }

  function handleChange(index: number, raw: string) {
    const char = raw.replace(/\D/g, '').slice(-1);
    setDigit(index, char || ' ');
    if (char && index < length - 1) refs.current[index + 1]?.focus();
  }

  function handleKeyDown(index: number, e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Backspace' && !digits[index].trim() && index > 0) refs.current[index - 1]?.focus();
  }

  function handlePaste(e: React.ClipboardEvent<HTMLInputElement>) {
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, length);
    if (pasted) {
      e.preventDefault();
      onChange(pasted.padEnd(length, '').slice(0, length).replace(/\s/g, ''));
      refs.current[Math.min(pasted.length, length - 1)]?.focus();
    }
  }

  return (
    <div role="group" aria-label={`${length}-digit verification code`} style={{ display: 'flex', gap: 'var(--space-2)' }}>
      {digits.map((d, i) => (
        <input
          key={i}
          ref={(el) => (refs.current[i] = el)}
          value={d.trim()}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          onPaste={handlePaste}
          inputMode="numeric"
          maxLength={1}
          aria-label={`Digit ${i + 1} of ${length}`}
          style={{ width: 52, height: 56, textAlign: 'center', font: '600 20px var(--font-mono)', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', background: 'var(--surface-card)', color: 'var(--text-primary)' }}
        />
      ))}
    </div>
  );
}
