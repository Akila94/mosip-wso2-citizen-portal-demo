/**
 * The SPA's only route to the network.
 *
 * Every call is **same-origin** to a `/bff/…` path served by citizen-portal-bff.
 * The browser never talks to WSO2 IS or to gov-services-api directly, and this
 * module never sets an `Authorization` header — it cannot, because the SPA has
 * no token to put in one. Access and ID tokens live exclusively in the BFF's
 * server-side session; the browser holds only an opaque, HttpOnly session
 * cookie. That is the whole point of the Backend-for-Frontend pattern and the
 * property to check in devtools when demoing (see PORTAL-INTEGRATION-PLAN.md's
 * verification step 10).
 *
 * Errors are normalised to a plain `Error(message)`, which is the shape
 * `hooks/useAsync.ts` already expects, so every screen's existing
 * loading/error/retry handling keeps working unchanged.
 */
import { CLIENTS, type AppKey } from '../config/clients';

/** Longest error text accepted from a response before it is truncated. */
const MAX_ERROR_TEXT = 300;

/**
 * An HTTP-level failure, carrying the status so callers can distinguish
 * "signed out" (401) from a genuine service fault. `message` is always
 * already human-readable — screens render it directly.
 */
export class HttpError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'HttpError';
    this.status = status;
  }
}

/**
 * The per-app CSRF tokens, held in memory only.
 *
 * The BFF sets a CSRF cookie scoped to `Path=/bff/{app}`, which the browser
 * dutifully sends on every request to that app's endpoints — but which
 * `document.cookie` cannot read from a page at `/` or `/apps/…`, because a
 * cookie is only visible to documents whose path matches its own (RFC 6265
 * §5.4). The BFF therefore also returns the token in the body of
 * `GET /bff/{app}/session`, and `AuthContext` hands it to us here.
 *
 * This is still a genuine double-submit: the server compares the header we
 * send against the cookie the browser sends, and a cross-origin attacker can
 * cause the cookie to be sent but can read neither the cookie nor the JSON
 * body that carries the token.
 */
const csrfTokens = new Map<AppKey, string>();

/** Records (or, with null, forgets) an app's CSRF token. Called by AuthContext. */
export function setCsrfToken(appKey: AppKey, token: string | null): void {
  if (token) csrfTokens.set(appKey, token);
  else csrfTokens.delete(appKey);
}

/**
 * Builds a same-origin URL under an app's BFF namespace.
 *
 * `path` must be a rooted, relative path. Rejecting anything else keeps a
 * caller from turning a data call into a request to another origin — the
 * client-side half of the unvalidated-redirect guard the BFF also enforces
 * server-side (WSO2 secure engineering guidelines §1.22).
 */
function bffUrl(appKey: AppKey, path: string, params?: Record<string, string | number | undefined>): string {
  if (!path.startsWith('/') || path.startsWith('//')) {
    throw new Error(`http: refusing to build a request for a non-relative path: ${path}`);
  }
  const url = CLIENTS[appKey].bffBase + path;
  if (!params) return url;

  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined) query.set(key, String(value));
  }
  const encoded = query.toString();
  return encoded ? `${url}?${encoded}` : url;
}

/**
 * Turns a failed response into a message worth showing a citizen.
 *
 * The BFF answers some failures with JSON (`{"error": "…"}`) and others with
 * Go's plain-text `http.Error`, so both are handled. Whatever comes back is
 * length-capped: an error body is remote input, and an unbounded one has no
 * business being pasted into the UI.
 */
async function errorMessageFor(response: Response): Promise<string> {
  switch (response.status) {
    case 401:
      return 'Your session has expired. Sign in again to continue.';
    case 403:
      return 'That action was refused. Refresh the page and try again.';
    case 502:
    case 503:
    case 504:
      return 'The government service is temporarily unavailable. Please try again.';
    default:
      break;
  }

  let detail = '';
  try {
    const text = (await response.text()).trim();
    if (text.startsWith('{')) {
      const parsed: unknown = JSON.parse(text);
      if (parsed && typeof parsed === 'object') {
        const record = parsed as Record<string, unknown>;
        const candidate = record.error ?? record.message;
        if (typeof candidate === 'string') detail = candidate;
      }
    } else {
      detail = text;
    }
  } catch {
    // A body that is unreadable or not JSON after all is not itself worth
    // reporting — the status code below still tells the citizen something.
    detail = '';
  }

  detail = detail.slice(0, MAX_ERROR_TEXT).trim();
  return detail || `Request failed (${response.status}).`;
}

async function request<T>(appKey: AppKey, path: string, init: RequestInit, params?: Record<string, string | number | undefined>): Promise<T> {
  let response: Response;
  try {
    response = await fetch(bffUrl(appKey, path, params), {
      // Send this app's own path-scoped session cookie, and nothing
      // cross-origin — the SPA has no cross-origin calls to make.
      credentials: 'same-origin',
      redirect: 'follow',
      ...init,
      headers: { Accept: 'application/json', ...(init.headers ?? {}) },
    });
  } catch (cause) {
    // fetch() rejects only on a genuine network/transport failure.
    throw new Error('Could not reach the service. Check your connection and try again.', { cause });
  }

  if (!response.ok) {
    throw new HttpError(response.status, await errorMessageFor(response));
  }

  if (response.status === 204) return undefined as T;

  try {
    return (await response.json()) as T;
  } catch (cause) {
    throw new Error('The service returned an unreadable response.', { cause });
  }
}

/** GET a JSON resource from this app's `/bff/{app}/api` namespace. */
export function apiGet<T>(appKey: AppKey, path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  return request<T>(appKey, `/api${path}`, { method: 'GET' }, params);
}

/**
 * GET a genuinely public resource from this app's `/bff/{app}/public`
 * namespace — no session required, and none created.
 *
 * Kept as its own function rather than a flag on `apiGet` so that every public
 * call site is greppable: an endpoint reachable without authentication is a
 * deliberate decision each time, not a default anyone can drift into.
 */
export function publicGet<T>(appKey: AppKey, path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  return request<T>(appKey, `/public${path}`, { method: 'GET' }, params);
}

/**
 * POST to this app's `/bff/{app}/api` namespace, carrying the CSRF token the
 * BFF requires on every state-changing route. A missing token is failed here
 * rather than sent and rejected server-side, so the cause is obvious.
 *
 * Declared `async` deliberately: the missing-token case must reject the
 * returned promise rather than throw synchronously, or a caller that invokes
 * this from inside a React effect would blow up the render instead of
 * surfacing an error state.
 */
export async function apiPost<T>(appKey: AppKey, path: string, body?: unknown): Promise<T> {
  const csrfToken = csrfTokens.get(appKey);
  if (!csrfToken) {
    throw new HttpError(401, 'Your session has expired. Sign in again to continue.');
  }
  return request<T>(appKey, `/api${path}`, {
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
}

/** The session projection the BFF returns from `GET /bff/{app}/session`. */
export interface SessionResponse {
  authenticated: boolean;
  clientId?: string;
  appName?: string;
  appKey: string;
  user: {
    sub: string;
    name?: string;
    givenName?: string;
    familyName?: string;
    email?: string;
    phoneNumber?: string;
    birthdate?: string;
    picture?: string;
  };
  assuranceLevel: string;
  idp?: string;
  sid: string;
  acr?: string;
  amr?: string[];
  authTime?: number;
  expiresAt?: number;
  csrfToken?: string;
}

/**
 * Reads the current session. A 401 is the normal "not signed in" answer, not
 * an error, so it resolves to null rather than throwing — the caller is the
 * bootstrap that decides what to do about it.
 */
export async function fetchSession(appKey: AppKey): Promise<SessionResponse | null> {
  try {
    return await request<SessionResponse>(appKey, '/session', { method: 'GET' });
  } catch (err) {
    if (err instanceof HttpError && err.status === 401) return null;
    throw err;
  }
}

/**
 * Ends the BFF's own session and returns the WSO2 IS RP-initiated logout URL
 * the browser must then navigate to. Ending the IS session is what triggers
 * back-channel logout to every other app, so the caller must actually follow
 * this URL — dropping it would leave the IS session alive and the next
 * "sign in" would be silent.
 */
export async function postLogout(appKey: AppKey): Promise<{ logoutUrl: string }> {
  const csrfToken = csrfTokens.get(appKey);
  return request<{ logoutUrl: string }>(appKey, '/logout', {
    method: 'POST',
    headers: csrfToken ? { 'X-CSRF-Token': csrfToken } : {},
  });
}

/**
 * The BFF login URL to navigate to, carrying where to come back to.
 *
 * `returnTo` is forced to a rooted relative path here, and validated again by
 * the BFF against this app's own route prefix before it is ever used as a
 * redirect target. Two independent checks, because an unvalidated redirect is
 * exactly the kind of thing that survives one.
 */
export function loginUrl(appKey: AppKey, returnTo: string): string {
  const safeReturnTo = returnTo.startsWith('/') && !returnTo.startsWith('//') ? returnTo : CLIENTS[appKey].routeBase;
  return `${CLIENTS[appKey].bffBase}/login?returnTo=${encodeURIComponent(safeReturnTo)}`;
}
