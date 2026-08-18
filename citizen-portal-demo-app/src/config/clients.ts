/**
 * The one place that knows which registered OIDC application each part of
 * the portal belongs to.
 *
 * Before this existed, the same facts were hardcoded independently in four
 * places — `VehicleRevenueLicenceScreen`'s `NARROW_IDENTITY` literal,
 * `applicationService`'s `verifiedIdentity` block, `sessionInspectorService`'s
 * `byClient` keys, and `portalService`'s consent fixtures — which is
 * inconsistency #1 and #7 in this app's own README. Everything that needs to
 * know "which app am I, and which BFF namespace do I talk to" now reads it
 * from here.
 *
 * Note what is deliberately *absent*: no client ID, no client secret, no
 * issuer URL, no scope the browser could act on. Those live only in the BFF's
 * environment. The SPA calls same-origin `/bff/…` paths and nothing else, so
 * no OIDC credential is ever compiled into this bundle — the main practical
 * argument for the BFF pattern, and the reason this file is safe to ship to a
 * browser.
 */

/** The three registered applications, keyed the same way the BFF keys them. */
export type AppKey = 'portal' | 'driving-licence' | 'revenue-licence';

export interface ClientRegistryEntry {
  appKey: AppKey;
  /**
   * Human-readable name exactly as registered in the WSO2 IS Console. Shown
   * as the "Released to" line in verified-identity panels and as the session
   * inspector's client label, so it must match what IS actually issued the
   * token to.
   */
  clientName: string;
  /** Owning agency, for display alongside clientName. */
  agency: string;
  /**
   * The catalogue service this application implements, or null for the portal
   * shell itself (which is not a catalogue entry). Used to route "start this
   * service" from a catalogue card into the right micro app.
   */
  serviceId: string | null;
  /**
   * This app's BFF namespace. Every call the SPA makes for this app is
   * same-origin under this prefix, which is also the `Path` its session and
   * CSRF cookies are scoped to — the browser therefore never presents one
   * app's cookies to another app's endpoints.
   */
  bffBase: string;
  /** The SPA route prefix this app owns, and the base for any returnTo. */
  routeBase: string;
  /**
   * Scopes as registered for this client in the IS Console. Display-only
   * metadata: the SPA cannot request or change a scope — IS decides what a
   * token carries, and the BFF holds the configured scope string. This copy
   * exists so consent-related wireframe screens can show a plausible,
   * per-application scope list without inventing one per screen. If the
   * Console registration changes, this list is documentation that has drifted,
   * never a functional bug.
   */
  scopes: string[];
}

export const CLIENTS: Record<AppKey, ClientRegistryEntry> = {
  portal: {
    appKey: 'portal',
    clientName: 'Citizen Portal',
    agency: 'MaroliaGov',
    serviceId: null,
    bffBase: '/bff/portal',
    routeBase: '/',
    scopes: ['openid', 'profile', 'email'],
  },
  'driving-licence': {
    appKey: 'driving-licence',
    clientName: 'Driving Licence Service',
    agency: 'Dept. of Motor Traffic',
    serviceId: 'svc-dl',
    bffBase: '/bff/driving-licence',
    routeBase: '/apps/driving-licence',
    scopes: ['openid', 'profile', 'email', 'address'],
  },
  'revenue-licence': {
    appKey: 'revenue-licence',
    clientName: 'Vehicle Revenue Licence',
    agency: 'Provincial Revenue Office',
    serviceId: 'svc-vrl',
    bffBase: '/bff/revenue-licence',
    routeBase: '/apps/revenue-licence',
    scopes: ['openid', 'profile', 'email'],
  },
};

export const CLIENT_LIST: ClientRegistryEntry[] = Object.values(CLIENTS);

/**
 * The micro app that implements a given catalogue service, if any. Services
 * with no micro app (every STUB entry in the catalogue) return undefined —
 * the caller keeps the citizen on the service-detail page rather than
 * navigating into an application that does not exist.
 */
export function clientForService(serviceId: string): ClientRegistryEntry | undefined {
  return CLIENT_LIST.find((client) => client.serviceId === serviceId);
}
