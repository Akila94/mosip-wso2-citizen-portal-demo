import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { AssuranceLevel } from '../services/types';

export interface AuthUser {
  id: string;
  displayName: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: AuthUser | null;
  assuranceLevel: AssuranceLevel;
  /** Mock sign-in for the still-simulated providers (local, MOSIP, National
   * Digital ID). Sets local state directly — no network call. */
  signIn: (level?: AssuranceLevel) => void;
  /** Real sign-in: full-page redirect to the BFF, which redirects to
   * ThunderID. Returns the browser to `returnTo` (default: current path)
   * once the session is established — this function never returns. */
  signInWithWallet: (returnTo?: string) => void;
  raiseAssurance: (level: AssuranceLevel) => void;
  signOut: () => void;
}

interface SessionResponse {
  authenticated: boolean;
  user?: {
    sub: string;
    name?: string;
    givenName?: string;
    familyName?: string;
  };
  assuranceLevel?: AssuranceLevel;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

/**
 * Two auth paths share this context: the still-simulated providers (local,
 * MOSIP, National Digital ID) just flip local state via `signIn`, matching
 * the pre-integration mock. `signInWithWallet` is the one real path — a
 * full-page redirect through the BFF to ThunderID; on return, the session
 * check below picks up the real cookie-backed session. Both converge on the
 * same isAuthenticated/user/assuranceLevel shape, so nothing downstream
 * needs to know which path authenticated the citizen.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [assuranceLevel, setAssuranceLevel] = useState<AssuranceLevel>('none');
  const [realUser, setRealUser] = useState<AuthUser | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch('/bff/portal/session', { credentials: 'include' })
      .then((res) => res.json())
      .then((data: SessionResponse) => {
        if (cancelled || !data.authenticated) return;
        setIsAuthenticated(true);
        setAssuranceLevel(data.assuranceLevel ?? 'substantial');
        setRealUser({
          id: data.user?.sub ?? 'citizen',
          displayName: data.user?.name ?? [data.user?.givenName, data.user?.familyName].filter(Boolean).join(' ') ?? 'Citizen',
        });
      })
      .catch(() => {
        // BFF unreachable or no session — stay signed out, same as before
        // this check existed.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const user: AuthUser | null = realUser ?? (isAuthenticated ? { id: 'demo-citizen-1', displayName: 'J. Doe' } : null);

  const value = useMemo<AuthState>(
    () => ({
      isAuthenticated,
      user,
      assuranceLevel,
      signIn: (level: AssuranceLevel = 'substantial') => {
        setIsAuthenticated(true);
        setAssuranceLevel(level);
      },
      signInWithWallet: (returnTo: string = window.location.pathname) => {
        window.location.href = `/bff/portal/login?returnTo=${encodeURIComponent(returnTo)}`;
      },
      raiseAssurance: (level: AssuranceLevel) => {
        setAssuranceLevel(level);
      },
      signOut: () => {
        if (realUser) {
          // Real session: RP-initiated logout at the BFF, which ends the
          // ThunderID session too.
          window.location.href = '/bff/portal/logout';
          return;
        }
        setIsAuthenticated(false);
        setAssuranceLevel('none');
      },
    }),
    [isAuthenticated, assuranceLevel, user, realUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}
