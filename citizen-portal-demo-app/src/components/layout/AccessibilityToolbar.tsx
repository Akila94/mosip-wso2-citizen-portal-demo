import React from 'react';
import { Button } from '../../design-system/components/core/Button';

export interface AccessibilityToolbarProps {
  variant?: 'compact' | 'full';
}

/**
 * Visual-only stub, by product decision: matches the wireframe's controls
 * (font scale, contrast, sign-language support) with no behavior wired up
 * yet. No design-system equivalent exists for these controls — flag for a
 * dedicated a11y-controls component once real behavior is specified.
 */
export function AccessibilityToolbar({ variant = 'full' }: AccessibilityToolbarProps) {
  const items = variant === 'compact' ? ['A− / A+', 'High contrast'] : ['A− / A+', 'High contrast', 'Sign-language support'];
  return (
    <div role="group" aria-label="Accessibility settings" style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
      {items.map((label) => (
        <Button key={label} type="button" variant="secondary" size={variant === 'compact' ? 'sm' : 'sm'} onClick={() => {}}>
          {label}
        </Button>
      ))}
    </div>
  );
}
