/**
 * Micro app A (Driving Licence Service), now backed by the real stack: the SPA
 * calls `/bff/driving-licence/api/…`, the BFF attaches **Application A's own**
 * access token, and `gov-services-api` accepts it only because its audience
 * matches that router. The Vehicle Revenue Licence's token is genuinely
 * rejected by these same endpoints — the audience story this demo exists to
 * show.
 *
 * One function is still an in-memory fixture — `getMedicalReviewError` —
 * because PORTAL-INTEGRATION-PLAN.md's Component 2 defines which routers
 * `gov-services-api` exposes and a medical-review router is not among them.
 */
import type {
  ApplicationConfig,
  ApplicationConfirmation,
  ApplicationErrorInfo,
  CitizenProfile,
  ServiceRequestOptions,
  TestDay,
} from './types';
import { apiGet, apiPost } from './http';
import { CLIENTS } from '../config/clients';
import { toVerifiedIdentity } from './verifiedIdentity';

const DL = 'driving-licence' as const;

const DEFAULT_DELAY_MS = 500;

/** Latency simulation, retained only for the one fixture-backed call below. */
function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Could not reach the application service. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

/**
 * Still a fixture: there is no medical-review endpoint to call. The screen it
 * feeds is reached by answering "yes" to the blackouts declaration, which is
 * itself client-side wizard state.
 */
const medicalReviewError: ApplicationErrorInfo = {
  title: "We can't continue online — a medical review is needed first",
  description: 'Your application is saved. It is not refused.',
  reasonHeading: 'The specific reason',
  reasonBody: 'You declared a history of blackouts. Regulation 14(3) requires a specialist assessment before a class B licence can be issued, whatever the rest of the application says.',
  reference: 'ref DL-2026-004871 · saved 14:38 · valid 90 days',
  routes: [
    { label: 'Book a specialist assessment', description: 'Any state clinic or approved specialist. The report is uploaded against this reference — you do not restart the application.' },
    { label: 'Continue without class B', description: 'Class A1 motorcycle is unaffected by regulation 14(3). You can amend step 1 and proceed today.' },
    { label: 'Talk to someone', description: 'Assisted agents at 42 service centres, or call 1500. Quote reference DL-2026-004871.' },
  ],
};

export const applicationService = {
  /**
   * The application's reference data, with the verified-identity block
   * replaced by the citizen's real registry record.
   *
   * The two calls are independent, so they run concurrently: the config is
   * static per-application reference data (licence classes, fee schedule), and
   * the identity is the dynamic, subject-keyed government record. Merging them
   * here rather than in each of the four step screens means no screen changed
   * when this stopped being a fixture.
   */
  async getApplicationConfig(): Promise<ApplicationConfig> {
    const [config, profile] = await Promise.all([
      apiGet<ApplicationConfig>(DL, '/config'),
      apiGet<CitizenProfile>(DL, '/identity'),
    ]);
    return { ...config, verifiedIdentity: toVerifiedIdentity(profile, CLIENTS[DL].clientName) };
  },

  /** A week of driving-test slots, `weekOffset` weeks out. */
  getTestWeek(weekOffset: number): Promise<TestDay[]> {
    return apiGet<TestDay[]>(DL, '/test-slots', { week: weekOffset });
  },

  /** Submits the application. CSRF-protected; see `http.ts`'s `apiPost`. */
  submitApplication(payload: unknown): Promise<ApplicationConfirmation> {
    return apiPost<ApplicationConfirmation>(DL, '/applications', payload);
  },

  /** MOCKED — no backing endpoint; see the `medicalReviewError` comment above. */
  getMedicalReviewError(opts?: ServiceRequestOptions) {
    return simulate(medicalReviewError, opts);
  },
};
