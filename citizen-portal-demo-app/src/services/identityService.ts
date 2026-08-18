/**
 * Wireframe B's data.
 *
 * **Everything in this file is a mock, and that is now the point.** WSO2
 * Identity Server hosts the real login, provider choice and consent — the
 * screens these functions feed live under `/wireframes/*` and exist to narrate
 * what IS does, not to do it. The three sign-in functions that used to fake an
 * authentication (`signInLocal`, `signInFederated`,
 * `completeFederatedVerification`) have been removed outright rather than left
 * behind a "demo only" comment: a function that flips this app into a
 * signed-in state without a real IS session would contradict the actual
 * session and make every subsequent data call fail with a confusing 401.
 *
 * `getServiceDetail` is the one mock here that still feeds a *real* route
 * (`/services/:serviceId`) — `gov-services-api` has no service-detail router,
 * so there is nothing real to call.
 */
import type {
  ServiceDetail,
  IdentityProvider,
  ConsentAttributeRequest,
  ServiceRequestOptions,
  AuthResult,
  AssuranceLevel,
} from './types';
import { clientForService } from '../config/clients';

const DEFAULT_DELAY_MS = 550;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Identity service temporarily unavailable. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

const drivingLicenceDetail: ServiceDetail = {
  id: 'svc-dl',
  title: 'Apply for a driving licence',
  category: 'Transport',
  department: 'Dept. of Motor Traffic',
  summary:
    'Apply for a new licence, renew an existing one, or replace a lost licence. Covers motorcycle, light vehicle, lorry and bus classes.',
  sections: [
    {
      id: 'what', heading: "What this service does", note: '', items: [
        'Issues a first licence, renews an existing one, or replaces a lost or damaged licence.',
        'Covers motorcycle (A1, A), light tricycle (B1), motor car (B), light lorry (C1) and light bus (D1) classes.',
        'Includes booking the written and practical test where the class requires one.',
      ],
    },
    {
      id: 'who', heading: "Who it's for", note: 'beneficiaries', items: [
        'Citizens and permanent residents of Marolia aged 18 or over (16 for A1 motorcycle).',
        'Holders of a valid learner permit for the class applied for.',
        'Existing licence holders renewing within 6 months of expiry.',
      ],
    },
    {
      id: 'eligibility', heading: 'Eligibility conditions', note: 'stated before the form, never inside it', items: [
        'You hold a learner permit for each class you apply for, issued at least 30 days ago.',
        'You have a medical fitness certificate issued within the last 3 months.',
        'You have no active court-ordered driving suspension.',
        'Your national identity record is verified — a Basic account will be asked to verify.',
      ],
    },
    {
      id: 'documents', heading: "Documents you'll need", note: 'upload specs shown before you start', items: [
        'Medical fitness certificate — PDF or JPG, max 5 MB, issued within 3 months.',
        'Learner permit — PDF or JPG, max 5 MB, both sides in one file.',
        'Photograph — not required: taken from your verified national ID record.',
      ],
    },
    {
      id: 'processing', heading: 'How the application is processed', note: '', items: [
        'Submitted applications are checked within 3 working days; you are notified in My Timeline.',
        'Test results are recorded against your application; the licence is issued to My Documents.',
        'Collection is at the pick-up station you choose in step 2, or by post for renewals.',
      ],
    },
  ],
  channels: [
    { id: 'app', title: 'Mobile app', description: 'MaroliaGov app, iOS and Android. Same account.' },
    { id: 'ussd', title: 'USSD short code', description: 'Dial *363# for status and payment. No smartphone needed.' },
    { id: 'agent', title: 'Assisted agent', description: 'An accredited agent completes the form with you.' },
    { id: 'centre', title: 'Service centre', description: '42 counters nationwide. Walk-in or booked.' },
  ],
  cost: { fee: '$18', processingTime: '10 working days', formLength: '4 steps · ~12 min', payment: 'now or within 7 days' },
  signInWith: [
    'National Digital ID — the fastest route; your details are filled from your verified record.',
    'Mobile number with one-time code.',
    'A MaroliaGov username and password.',
    'A MOSIP-based national eID — you will be asked to verify your identity before submitting.',
  ],
};

/** Generic fallback so any catalogue card is clickable even before its own
 * detail copy is authored — real content should replace this per service. */
function genericDetail(id: string, title: string, description: string, fee: string, timeEstimate: string, category: string): ServiceDetail {
  return {
    id,
    title,
    category,
    department: `${category} authority`,
    summary: description,
    sections: [
      { id: 'what', heading: 'What this service does', note: '', items: [description] },
      { id: 'eligibility', heading: 'Eligibility conditions', note: 'stated before the form, never inside it', items: ['Details for this service are being finalised.'] },
    ],
    channels: [
      { id: 'app', title: 'Mobile app', description: 'MaroliaGov app, iOS and Android. Same account.' },
      { id: 'centre', title: 'Service centre', description: '42 counters nationwide. Walk-in or booked.' },
    ],
    cost: { fee, processingTime: 'varies', formLength: `~${timeEstimate}`, payment: 'now or within 7 days' },
    signInWith: ['National Digital ID — the fastest route; your details are filled from your verified record.', 'A MaroliaGov username and password.'],
  };
}

/**
 * The providers WSO2 IS's own login page offers. Mirrors the Login Flow
 * configured in the Console (Username & Password + MOSIP eSignet); this copy
 * exists only for the wireframe screen, which no longer performs a sign-in.
 */
const identityProviders: IdentityProvider[] = [
  { id: 'mosip', name: 'Continue with MOSIP', description: 'national eID · substantial assurance · external hand-off', assuranceLevel: 'substantial', externalHop: true },
  { id: 'national-digital-id', name: 'Continue with National Digital ID', description: 'national eID · substantial assurance · external hand-off', assuranceLevel: 'substantial', externalHop: true },
  { id: 'mobile-otp', name: 'Sign in with Mobile OTP', description: 'code sent by SMS · basic assurance', assuranceLevel: 'basic', externalHop: false },
];

/**
 * Attributes a service asks consent for, keyed by attribute id. Which subset
 * a given service requests is derived from that service's registered scopes
 * in `config/clients.ts`, so the two micro apps genuinely differ here rather
 * than both showing one hardcoded list.
 */
const CONSENT_ATTRIBUTES: Record<string, ConsentAttributeRequest> = {
  name: { id: 'name', label: 'Full name', required: true, value: 'from your verified record', source: 'National Digital ID' },
  email: { id: 'email', label: 'Email address', required: false, value: 'from your verified record', source: 'National Digital ID' },
  nic: { id: 'nic', label: 'NIC number', required: true, value: 'from your verified record', source: 'National Digital ID' },
  dob: { id: 'dob', label: 'Date of birth', required: true, value: 'from your verified record', source: 'National Digital ID' },
  address: { id: 'address', label: 'Address', required: true, value: 'from your verified record', source: 'National Digital ID' },
};

/** Which consent attributes each OIDC scope covers. */
const SCOPE_ATTRIBUTES: Record<string, string[]> = {
  profile: ['name', 'dob'],
  email: ['email'],
  address: ['nic', 'address'],
};

export const identityService = {
  /** MOCKED — `gov-services-api` has no service-detail router. */
  getServiceDetail(serviceId: string, opts?: ServiceRequestOptions) {
    if (serviceId === 'svc-dl') return simulate(drivingLicenceDetail, opts);
    return simulate(genericDetail(serviceId, 'Service detail', 'Service description not yet authored for this catalogue entry.', '$—', '~5 min', 'Government'), opts);
  },

  /** MOCKED — WSO2 IS's own login page is the real list. */
  getIdentityProviders(opts?: ServiceRequestOptions) {
    return simulate(identityProviders, opts);
  },

  /**
   * MOCKED — IS hosts the real consent page. Now honours `serviceId`: the
   * attributes shown are derived from the scopes registered for whichever
   * application implements that service, so the Vehicle Revenue Licence asks
   * for visibly less than the Driving Licence Service does.
   */
  getConsentAttributes(serviceId: string, opts?: ServiceRequestOptions) {
    const client = clientForService(serviceId);
    const attributeIds = new Set<string>();
    for (const scope of client?.scopes ?? ['profile']) {
      for (const id of SCOPE_ATTRIBUTES[scope] ?? []) attributeIds.add(id);
    }
    const attributes = [...attributeIds].map((id) => CONSENT_ATTRIBUTES[id]).filter(Boolean);
    return simulate(attributes, opts);
  },

  /** MOCKED — IS records the real consent decision. */
  submitConsent(_serviceId: string, _decisions: Record<string, boolean>, opts?: ServiceRequestOptions) {
    return simulate({ granted: true }, opts);
  },

  /**
   * MOCKED — the wireframe step-up screen's 6-digit code check. Any 6-digit
   * code succeeds.
   *
   * The real step-up path (`GET /bff/{app}/step-up`, which re-authorizes
   * against IS with `prompt=login`) is implemented in the BFF but deliberately
   * not wired into this flow: a full-page round trip through IS mid-application
   * would discard the in-memory wizard state the citizen has just filled in.
   */
  submitStepUp(code: string, opts?: ServiceRequestOptions): Promise<AuthResult> {
    if (!/^\d{6}$/.test(code)) return Promise.reject(new Error('Enter the 6-digit code.'));
    return simulate({ assuranceLevel: 'substantial' as AssuranceLevel }, opts);
  },
};
