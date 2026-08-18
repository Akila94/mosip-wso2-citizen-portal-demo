import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { AssuranceLevel } from '../services/types';
import { CLIENTS, type AppKey } from '../config/clients';
import { fetchSession, loginUrl, postLogout, setCsrfToken, type SessionResponse } from '../services/http';

export interface AuthUser {
  /** Stable identifier for React keys and the like. This is the OIDC `sub`. */
  id: string;
  /** What to show in the header. Never the `sub` — see `displayNameFor`. */
  displayName: string;
  /**
   * The OIDC subject: eSignet mints this as a *pairwise pseudonymous
   * identifier* (PSUT), unique per relying party. It is deliberately not a
   * national ID number and must never be presented as one — the NIC on screen
   * comes from the government registry, looked up server-side against this
   * value. Shown verbatim only in the session inspector, which is a technical
   * demo tool.
   */
  sub: string;
  name?: string;
  givenName?: string;
  familyName?: string;
  email?: string;
  phoneNumber?: string;
  birthdate?: string;
  /** Usually a `data:` URI supplied by eSignet, not an external URL. */
  picture?: string;
}

export interface AuthState {
  /** Which registered application this subtree's session belongs to. */
  appKey: AppKey;
  isAuthenticated: boolean;
  /**
   * True until the initial `GET /bff/{app}/session` settles. Screens should
   * render a placeholder rather than a signed-out state while this is true,
   * otherwise every page load flashes "Sign in" before the real session
   * arrives.
   */
  isLoading: boolean;
  user: AuthUser | null;
  assuranceLevel: AssuranceLevel;
  /** Identity provider that actually authenticated the citizen, as reported by the BFF. */
  idp: string | null;
  /** Name of this app's OIDC client, for "Released to" lines. */
  clientName: string;
  /** Redirects the browser to WSO2 IS to sign in, returning to `returnTo`. */
  signIn: (returnTo?: string) => void;
  /** Ends every session — this app's, the BFF's, and the IS session behind them. */
  signOut: () => void;
  raiseAssurance: (level: AssuranceLevel) => void;
  /** Re-reads the session from the BFF. */
  reload: () => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

const ASSURANCE_LEVELS: readonly AssuranceLevel[] = ['none', 'basic', 'substantial'];

/**
 * Narrows the BFF's assurance string to the union the UI switches on. An
 * unrecognised value is treated as `none` rather than passed through: this is
 * a value that gates what the UI offers, so an unknown one must fail closed.
 */
function toAssuranceLevel(value: string | undefined): AssuranceLevel {
  return ASSURANCE_LEVELS.find((level) => level === value) ?? 'none';
}

/**
 * Picks something human to call the citizen, in descending order of how much
 * it looks like a name. Falls back to a neutral word rather than to `sub`,
 * which is an opaque pairwise identifier and would be both meaningless and
 * inappropriate to display.
 */
function displayNameFor(user: SessionResponse['user']): string {
  const fullName = user.name?.trim();
  if (fullName) return fullName;

  const composed = [user.givenName, user.familyName].filter(Boolean).join(' ').trim();
  if (composed) return composed;

  return user.email?.trim() || 'Citizen';
}

function toAuthUser(session: SessionResponse): AuthUser {
  return {
    id: session.user.sub,
    displayName: displayNameFor(session.user),
    sub: session.user.sub,
    name: session.user.name,
    givenName: session.user.givenName,
    familyName: session.user.familyName,
    email: session.user.email,
    phoneNumber: session.user.phoneNumber,
    birthdate: session.user.birthdate,
    picture: session.user.picture,
  };
}

/**
 * Holds one registered application's authenticated session, read from that
 * app's own BFF namespace.
 *
 * There is one provider per route tree — the portal shell at the root, and a
 * nested one inside each micro app — because the three apps are genuinely
 * separate OIDC clients with separate tokens, audiences and released claim
 * sets. A micro-app page therefore reads two sessions (its own and the
 * portal's, which still wraps the shell); that is not redundant work, it is
 * the thing the demo exists to show.
 *
 * No token of any kind reaches this component. `GET /bff/{app}/session`
 * returns a projection with no token field by construction, and the session
 * cookie behind it is HttpOnly and path-scoped to `/bff/{app}`.
 */
export function AuthProvider({ appKey, children }: { appKey: AppKey; children: React.ReactNode }) {
  const [session, setSession] = useState<SessionResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  /**
   * Presentation-only assurance override, set by the wireframe step-up screen.
   *
   * It cannot grant anything. Every server-side decision that depends on
   * assurance derives it from the verified session's `amr` claim and ignores
   * whatever a client says — `gov-services-api` has an explicit test for that
   * (`TestPortalCatalogueDerivesAssuranceLevelFromSessionNotRequest`). This
   * exists only so the wireframe step-up screen still has a visible effect
   * during the demo narrative, and it is dropped the moment the real session
   * is re-read, so it can never survive a reload.
   *
   * The real path — `GET /bff/{app}/step-up`, which re-authorizes against IS
   * with `prompt=login` — is implemented in the BFF but deliberately not
   * wired up here: a full-page round trip through IS would discard the
   * in-memory application wizard state mid-application.
   */
  const [assuranceOverride, setAssuranceOverride] = useState<AssuranceLevel | null>(null);

  const load = useCallback(async (signal?: { cancelled: boolean }) => {
    setIsLoading(true);
    try {
      const result = await fetchSession(appKey);
      if (signal?.cancelled) return;
      setSession(result);
      setAssuranceOverride(null);
      setCsrfToken(appKey, result?.csrfToken ?? null);
    } catch {
      // A session read that fails for any reason other than "not signed in"
      // (which fetchSession reports as null) leaves the citizen signed out
      // rather than in an indeterminate state. The failure surfaces on the
      // next data call, which has a screen to report it on.
      if (signal?.cancelled) return;
      setSession(null);
      setCsrfToken(appKey, null);
    } finally {
      if (!signal?.cancelled) setIsLoading(false);
    }
  }, [appKey]);

  useEffect(() => {
    const signal = { cancelled: false };
    void load(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [load]);

  const isAuthenticated = session?.authenticated === true;
  const user = useMemo(() => (session?.authenticated ? toAuthUser(session) : null), [session]);

  const signIn = useCallback(
    (returnTo?: string) => {
      const target = returnTo ?? window.location.pathname + window.location.search;
      window.location.assign(loginUrl(appKey, target));
    },
    [appKey]
  );

  const signOut = useCallback(() => {
    // Single logout: the BFF drops its own session and hands back the WSO2 IS
    // RP-initiated logout URL. Following it ends the IS session, which in turn
    // fires back-channel logout to all three registered apps — so one sign-out
    // really does end every micro app's session, not just this one's.
    void postLogout(appKey)
      .then(({ logoutUrl }) => {
        setCsrfToken(appKey, null);
        window.location.assign(logoutUrl);
      })
      .catch(() => {
        // Even if the BFF call failed, this browser should stop presenting a
        // signed-in UI. Re-reading the session settles what is actually true.
        setCsrfToken(appKey, null);
        void load();
      });
  }, [appKey, load]);

  const value = useMemo<AuthState>(
    () => ({
      appKey,
      isAuthenticated,
      isLoading,
      user,
      assuranceLevel: assuranceOverride ?? toAssuranceLevel(session?.assuranceLevel),
      idp: session?.idp ?? null,
      clientName: session?.appName ?? CLIENTS[appKey].clientName,
      signIn,
      signOut,
      raiseAssurance: setAssuranceOverride,
      reload: () => void load(),
    }),
    [appKey, isAuthenticated, isLoading, user, assuranceOverride, session, signIn, signOut, load]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}
