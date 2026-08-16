import type {
  ServiceDetail,
  IdentityProvider,
  ConsentAttributeRequest,
  AuthResult,
  ServiceRequestOptions,
  AssuranceLevel,
} from './types';

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

const identityProviders: IdentityProvider[] = [
  { id: 'mosip', name: 'Continue with MOSIP', description: 'national eID · substantial assurance · external hand-off', assuranceLevel: 'substantial', externalHop: true },
  { id: 'national-digital-id', name: 'Continue with National Digital ID', description: 'national eID · substantial assurance · external hand-off', assuranceLevel: 'substantial', externalHop: true },
  { id: 'mobile-otp', name: 'Sign in with Mobile OTP', description: 'code sent by SMS · basic assurance', assuranceLevel: 'basic', externalHop: false },
];

const consentAttributes: ConsentAttributeRequest[] = [
  { id: 'name', label: 'Full name', required: true, value: 'John Doe', source: 'National Digital ID' },
  { id: 'nic', label: 'NIC number', required: true, value: '19•• ••• •••• 4471', source: 'National Digital ID' },
  { id: 'dob', label: 'Date of birth', required: true, value: '04 Mar 1996', source: 'National Digital ID' },
  { id: 'address', label: 'Address', required: true, value: '14 Lake Road, Marolia City', source: 'National Digital ID' },
  { id: 'photo', label: 'Photograph', required: false, value: '[ image ]', source: 'National Digital ID' },
];

export const identityService = {
  getServiceDetail(serviceId: string, opts?: ServiceRequestOptions) {
    if (serviceId === 'svc-dl') return simulate(drivingLicenceDetail, opts);
    return simulate(genericDetail(serviceId, 'Service detail', 'Service description not yet authored for this catalogue entry.', '$—', '~5 min', 'Government'), opts);
  },

  getIdentityProviders(opts?: ServiceRequestOptions) {
    return simulate(identityProviders, opts);
  },

  getConsentAttributes(_serviceId: string, opts?: ServiceRequestOptions) {
    return simulate(consentAttributes, opts);
  },

  /** Local username/password (or mobile+OTP) sign-in hosted by WSO2 Identity
   * Server. Always succeeds in the demo; replace with a real OIDC
   * authorization-code exchange when the identity server is wired up. */
  signInLocal(_identifier: string, _secret: string, opts?: ServiceRequestOptions): Promise<AuthResult> {
    return simulate({ assuranceLevel: 'basic' as AssuranceLevel }, opts);
  },

  /** Kicks off federated sign-in. IdPs with externalHop=true resolve to a
   * pending state — the caller should route to the external IdP stub, then
   * call completeFederatedVerification. */
  signInFederated(idpId: string, opts?: ServiceRequestOptions): Promise<AuthResult> {
    const idp = identityProviders.find((p) => p.id === idpId);
    return simulate({ assuranceLevel: (idp?.assuranceLevel ?? 'basic') as AssuranceLevel }, opts);
  },

  /** Completes the hand-off from the external national eID stub (frame 9). */
  completeFederatedVerification(_idpId: string, opts?: ServiceRequestOptions): Promise<AuthResult> {
    return simulate({ assuranceLevel: 'substantial' as AssuranceLevel }, opts);
  },

  submitConsent(_serviceId: string, _decisions: Record<string, boolean>, opts?: ServiceRequestOptions) {
    return simulate({ granted: true }, opts);
  },

  /** Adaptive step-up MFA (TOTP or SMS code). Any 6-digit code succeeds in the demo. */
  submitStepUp(code: string, opts?: ServiceRequestOptions): Promise<AuthResult> {
    if (!/^\d{6}$/.test(code)) return Promise.reject(new Error('Enter the 6-digit code.'));
    return simulate({ assuranceLevel: 'substantial' as AssuranceLevel }, opts);
  },
};
