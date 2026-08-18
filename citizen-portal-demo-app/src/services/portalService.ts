/**
 * Wireframe A's data, now served by the real stack: the SPA calls
 * `/bff/portal/api/…`, the BFF attaches the portal client's access token, and
 * `gov-services-api` validates that token's signature, issuer, expiry and
 * audience before answering. The response shapes are unchanged, so every hook
 * and screen above this file is untouched.
 *
 * One function here is still an in-memory fixture — `getLifeEvents` — because
 * PORTAL-INTEGRATION-PLAN.md's Component 2 defines exactly which routers
 * `gov-services-api` exposes and life events are not among them. It is called
 * out individually below rather than left to be discovered.
 */
import type {
  LifeEvent,
  ServiceCategory,
  TimelineItem,
  AttributeRecord,
  DepartmentRecord,
  ConsentGrant,
  WalletDocument,
  AssuranceLevel,
  ServiceRequestOptions,
} from './types';
import { apiGet, publicGet } from './http';

const PORTAL = 'portal' as const;

/** How many timeline entries the landing page previews. */
const TIMELINE_PREVIEW_COUNT = 3;

const DEFAULT_DELAY_MS = 650;

/** Latency simulation, retained only for the one fixture-backed call below. */
function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Service temporarily unavailable. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

/**
 * Still a fixture: `gov-services-api` has no life-events router, so there is
 * nothing real to call. Kept mocked deliberately rather than inventing an
 * endpoint the design does not specify.
 */
const lifeEvents: LifeEvent[] = [
  { id: 'le-18', title: "I'm turning 18", serviceCount: '4 services' },
  { id: 'le-vehicle', title: 'I bought a vehicle', serviceCount: '5 services' },
  { id: 'le-move', title: "I'm moving house", serviceCount: '3 services' },
  { id: 'le-baby', title: 'I had a baby', serviceCount: '4 services' },
];

export const portalService = {
  /** MOCKED — no backing endpoint; see the `lifeEvents` comment above. */
  getLifeEvents(opts?: ServiceRequestOptions) {
    return simulate(lifeEvents, opts);
  },

  /**
   * The catalogue as a signed-out visitor sees it: every card in its own
   * published state, none promoted to READY or STEP_UP.
   *
   * A government service catalogue is public information, so this is served by
   * a genuinely unauthenticated endpoint rather than by a second copy of the
   * data living in the browser — one source of truth, and nothing here can
   * leak anything citizen-specific because the request carries no session and
   * no token at all.
   */
  getPublicServiceCatalogue() {
    return publicGet<ServiceCategory[]>(PORTAL, '/catalogue');
  },

  /**
   * The catalogue as an authenticated citizen sees it — cards flip from
   * "sign-in required" to READY or STEP_UP.
   *
   * `assuranceLevel` is **not** sent to the server, and the parameter is kept
   * only so callers re-fetch when assurance changes. The BFF derives assurance
   * itself from the verified session's `amr` claim and ignores anything the
   * client says, precisely so a client cannot self-report `substantial` and
   * unlock STEP_UP-gated entries it never stepped up for. Passing it here
   * would imply an influence it does not have.
   */
  getServiceCatalogue(_assuranceLevel: AssuranceLevel) {
    return apiGet<ServiceCategory[]>(PORTAL, '/catalogue');
  },

  getTimeline() {
    return apiGet<TimelineItem[]>(PORTAL, '/timeline');
  },

  /** Sliced client-side — the timeline endpoint has no separate preview form. */
  async getTimelinePreview() {
    const timeline = await apiGet<TimelineItem[]>(PORTAL, '/timeline');
    return timeline.slice(0, TIMELINE_PREVIEW_COUNT);
  },

  getAttributes() {
    return apiGet<AttributeRecord[]>(PORTAL, '/attributes');
  },

  getDepartmentRecords() {
    return apiGet<DepartmentRecord[]>(PORTAL, '/department-records');
  },

  /**
   * The apps holding a consent grant. Read-only: revoking a grant means
   * calling WSO2 IS's consent-management API, which is a separate integration
   * from this milestone's, so `gov-services-api` exposes no revoke route and
   * the UI's revoke control is disabled rather than wired to a write path the
   * server cannot honour.
   */
  getConsents() {
    return apiGet<ConsentGrant[]>(PORTAL, '/consents');
  },

  getWalletDocuments() {
    return apiGet<WalletDocument[]>(PORTAL, '/documents');
  },
};
