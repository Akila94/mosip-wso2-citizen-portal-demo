import React, { createContext, useContext, useMemo, useState } from 'react';
import type { AssuranceLevel } from '../services/types';

export interface AuthUser {
  id: string;
  displayName: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: AuthUser | null;
  assuranceLevel: AssuranceLevel;
  signIn: (level?: AssuranceLevel) => void;
  raiseAssurance: (level: AssuranceLevel) => void;
  signOut: () => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

/**
 * Demo-only mock session. Replace the internals of signIn/signOut with a
 * real WSO2 Identity Server OIDC flow (redirect + session/userinfo check)
 * when that integration lands — every component reads this context shape
 * (isAuthenticated / user / assuranceLevel), never a session object
 * directly, so nothing downstream needs to change.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [assuranceLevel, setAssuranceLevel] = useState<AssuranceLevel>('none');

  const user: AuthUser | null = isAuthenticated ? { id: 'demo-citizen-1', displayName: 'J. Doe' } : null;

  const value = useMemo<AuthState>(
    () => ({
      isAuthenticated,
      user,
      assuranceLevel,
      signIn: (level: AssuranceLevel = 'substantial') => {
        setIsAuthenticated(true);
        setAssuranceLevel(level);
      },
      raiseAssurance: (level: AssuranceLevel) => {
        setAssuranceLevel(level);
      },
      signOut: () => {
        // Single logout: in the real integration this ends the IS session
        // for every micro app sharing it, not just this shell.
        setIsAuthenticated(false);
        setAssuranceLevel('none');
      },
    }),
    [isAuthenticated, assuranceLevel]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}
