import React from 'react';
import { Button } from '../../design-system/components/core/Button';
import { Breadcrumb } from '../../design-system/components/navigation/Breadcrumb';
import { Placeholder } from '../common/Placeholder';
import { LanguageSwitcher } from './LanguageSwitcher';
import { AccessibilityToolbar } from './AccessibilityToolbar';
import { IdentityBadge } from './IdentityBadge';
import { AccountMenu } from './AccountMenu';
import { useAuth } from '../../context/AuthContext';
import type { Screen } from '../../App';

export interface AppHeaderProps {
  screen: Screen;
  onNavigate: (screen: Screen) => void;
  /** Trail after "Home", e.g. ["My Timeline"]. Landing page passes none. */
  breadcrumb?: string[];
  /** Header's Sign in button navigates to the identity login screen rather
   * than authenticating directly — the login options page is a real stop,
   * per the wireframe's own annotation ("Sign in" redirects to WSO2
   * Identity Server). Optional so screens without a generic sign-in
   * affordance don't need to pass it. */
  onSignIn?: () => void;
}

export function AppHeader({ onNavigate, breadcrumb, onSignIn }: AppHeaderProps) {
  const { isAuthenticated, assuranceLevel } = useAuth();
  return (
    <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 'var(--space-4)', padding: 'var(--space-3) var(--space-8)', borderBottom: '1px solid var(--border-default)', flexWrap: 'wrap' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)' }}>
        <Placeholder label="crest" width={40} height={40} />
        <button type="button" onClick={() => onNavigate('landing')} style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 0, display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: 2 }}>
          <span style={{ font: '700 16px var(--font-sans)', letterSpacing: '-0.01em', color: 'var(--text-primary)' }}>MaroliaGov</span>
          <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>Citizen services portal</span>
        </button>
        {breadcrumb && <Breadcrumb items={['Home', ...breadcrumb]} />}
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', flexWrap: 'wrap' }}>
        <LanguageSwitcher />
        {!isAuthenticated && !breadcrumb && <AccessibilityToolbar variant="compact" />}
        <IdentityBadge assuranceLevel={assuranceLevel} />
        {isAuthenticated ? <AccountMenu onNavigate={onNavigate} /> : <Button onClick={() => (onSignIn ? onSignIn() : onNavigate('identityLogin'))}>Sign in</Button>}
      </div>
    </header>
  );
}
