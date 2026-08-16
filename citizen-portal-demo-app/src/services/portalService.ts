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

const DEFAULT_DELAY_MS = 650;

/**
 * Demo-only latency + failure simulation. When the real MaroliaGov BFF
 * exists, replace the resolved payloads below with actual fetch() calls —
 * the exported shape of `portalService` should not need to change, so
 * hooks and components never have to change either.
 */
function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Service temporarily unavailable. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

function eff(signInRequired: boolean, signature: 'none' | 'optional' | 'required', steps: number, timeEstimate: string, fee: string) {
  return { signInRequired, signature, steps, timeEstimate, fee };
}

const lifeEvents: LifeEvent[] = [
  { id: 'le-18', title: "I'm turning 18", serviceCount: '4 services' },
  { id: 'le-vehicle', title: 'I bought a vehicle', serviceCount: '5 services' },
  { id: 'le-move', title: "I'm moving house", serviceCount: '3 services' },
  { id: 'le-baby', title: 'I had a baby', serviceCount: '4 services' },
];

const serviceCategories: (Omit<ServiceCategory, 'services'> & { services: (ServiceCategory['services'][number])[] })[] = [
  {
    id: 'transport', name: 'Transport', count: '3 services',
    services: [
      { id: 'svc-dl', title: 'Apply for a driving licence', description: 'New, renewal or duplicate licence for all classes.', state: 'LIVE', stepUpRequired: false, effort: eff(true, 'optional', 4, '12 min', 'fee $18') },
      { id: 'svc-vrl', title: 'Renew vehicle revenue licence', description: 'Annual revenue licence for a registered vehicle.', state: 'LIVE', stepUpRequired: false, effort: eff(true, 'none', 2, '4 min', 'fee from $15') },
      { id: 'svc-transfer', title: 'Transfer vehicle ownership', description: 'Record a change of registered owner.', state: 'STUB', stepUpRequired: true, effort: eff(true, 'required', 5, '20 min', 'fee $35') },
    ],
  },
  {
    id: 'health', name: 'Health', count: '2 services',
    services: [
      { id: 'svc-fitness', title: 'Request medical fitness certificate', description: 'Book a fitness assessment at a state clinic.', state: 'STUB', stepUpRequired: false, effort: eff(true, 'none', 3, '8 min', 'fee $8') },
      { id: 'svc-vaccine', title: 'Vaccination record', description: 'View and share your immunisation history.', state: 'STUB', stepUpRequired: false, effort: eff(true, 'none', 1, '1 min', 'free') },
    ],
  },
  {
    id: 'revenue', name: 'Revenue', count: '2 services',
    services: [
      { id: 'svc-tax', title: 'File individual income tax', description: 'Annual return for salaried and self-employed filers.', state: 'STUB', stepUpRequired: true, effort: eff(true, 'required', 6, '25 min', 'free') },
      { id: 'svc-clearance', title: 'Request tax clearance', description: 'Clearance letter for visa or tender purposes.', state: 'STUB', stepUpRequired: false, effort: eff(true, 'none', 2, '5 min', 'fee $11') },
    ],
  },
  {
    id: 'civil', name: 'Civil registration', count: '1 service',
    services: [
      { id: 'svc-birth', title: 'Order a birth certificate copy', description: 'Certified copy delivered or collected.', state: 'STUB', stepUpRequired: false, effort: eff(true, 'none', 3, '6 min', 'fee $5') },
    ],
  },
  {
    id: 'land', name: 'Land', count: '1 service',
    services: [
      { id: 'svc-land', title: 'Search land title', description: 'Public title search by deed or parcel number.', state: 'STUB', stepUpRequired: false, effort: eff(false, 'none', 1, '2 min', 'free') },
    ],
  },
];

const timelineFull: TimelineItem[] = [
  { id: 't1', date: '12 Sep 2026', relative: 'in 40 days', title: 'Driving licence expires', description: 'Class B · renewal opens 30 days before expiry', chip: 'Action needed', actionLabel: 'Renew' },
  { id: 't2', date: '30 Sep 2026', relative: 'next month', title: 'Vehicle revenue licence due — CAB-4471', description: 'Annual renewal · $15 + insurance check', chip: 'Payment due', actionLabel: 'Pay' },
  { id: 't3', date: '20 Aug 2026', relative: 'Thursday', title: 'Written test appointment', description: 'Colombo West licence office · 09:30 · bring NIC', chip: 'Appointment', actionLabel: 'View' },
  { id: 't4', date: '18 Aug 2026', relative: 'in 5 days', title: 'Driving licence application under review', description: 'Ref DL-2026-004871 · submitted 13 Aug', chip: 'Waiting on government', actionLabel: 'Track' },
  { id: 't5', date: '01 Nov 2026', relative: 'in 80 days', title: 'Passport expires', description: 'Renewal available 6 months before expiry', chip: 'For information', actionLabel: 'Remind me' },
];

const attributes: AttributeRecord[] = [
  { id: 'a1', label: 'Full name', value: 'John Doe', source: 'verified', sourceLabel: 'Verified — National Digital ID', editable: false },
  { id: 'a2', label: 'NIC number', value: '19•• ••• •••• 4471', source: 'verified', sourceLabel: 'Verified — National Digital ID', editable: false },
  { id: 'a3', label: 'Date of birth', value: '04 Mar 1996', source: 'verified', sourceLabel: 'Verified — National Digital ID', editable: false },
  { id: 'a4', label: 'Address', value: '14 Lake Road, Marolia City', source: 'verified', sourceLabel: 'Verified — National Digital ID', editable: false },
  { id: 'a5', label: 'Photograph', value: '[ image placeholder ]', source: 'verified', sourceLabel: 'Verified — National Digital ID', editable: false },
  { id: 'a6', label: 'Mobile number', value: '+94 7•• ••• 220', source: 'self-asserted', sourceLabel: 'Self-asserted — verified by OTP', editable: true },
  { id: 'a7', label: 'Email', value: 'john.doe@example.mr', source: 'self-asserted', sourceLabel: 'Self-asserted', editable: true },
  { id: 'a8', label: 'Blood group', value: 'O+', source: 'self-asserted', sourceLabel: 'Self-asserted', editable: true },
];

const departmentRecords: DepartmentRecord[] = [
  { id: 'r-transport', department: 'Transport', rows: [{ label: 'Driving licence', value: 'none yet' }, { label: 'Vehicles registered', value: '1 · CAB-4471' }, { label: 'Demerit points', value: '0' }] },
  { id: 'r-revenue', department: 'Revenue', rows: [{ label: 'Taxpayer ID', value: 'TIN-••• 8820' }, { label: 'Last return filed', value: '2025/26' }, { label: 'Outstanding', value: 'none' }] },
  { id: 'r-civil', department: 'Civil registration', rows: [{ label: 'Birth record', value: 'registered 1996' }, { label: 'Marital status', value: 'single' }, { label: 'Dependants', value: 'none' }] },
];

let consents: ConsentGrant[] = [
  { id: 'cons-dl', appName: 'Driving Licence Service', agency: 'Dept. of Motor Traffic', scopes: ['openid', 'profile', 'nic', 'dob', 'address', 'photograph'], grantedDate: '13 Aug 2026' },
  { id: 'cons-vrl', appName: 'Vehicle Revenue Licence', agency: 'Provincial Revenue Office', scopes: ['openid', 'profile', 'nic', 'vehicle_registry.read'], grantedDate: '13 Aug 2026' },
  { id: 'cons-birth', appName: 'Birth Certificate Service', agency: 'Registrar General', scopes: ['openid', 'profile', 'nic'], grantedDate: '02 Feb 2026' },
];

// "Demo start" state from the wireframe: the driving licence credential has
// not been issued yet. Its post-issuance state is the payoff of a later
// micro-app flow (Document C) — out of scope for this screen.
const walletDocuments: WalletDocument[] = [
  { id: 'doc-nic', title: 'National identity card', status: 'VALID', number: '19•• ••• •••• 4471', issuedDate: '11 Jan 2014', expiryDate: 'no expiry', primaryAction: 'Download', secondaryAction: 'Share' },
  { id: 'doc-dl', title: 'Driving licence', status: 'NOT_ISSUED', number: '—', issuedDate: '—', expiryDate: '—', primaryAction: 'Apply now', secondaryAction: 'Learn more' },
  { id: 'doc-vehicle', title: 'Vehicle registration — CAB-4471', status: 'VALID', number: 'CAB-4471', issuedDate: '02 Jun 2023', expiryDate: '30 Sep 2026', primaryAction: 'Download', secondaryAction: 'Share' },
];

export const portalService = {
  getLifeEvents(opts?: ServiceRequestOptions) {
    return simulate(lifeEvents, opts);
  },

  /** Cards flip from "sign-in required" to READY/STEP-UP once assurance is above 'none'. */
  getServiceCatalogue(assuranceLevel: AssuranceLevel, opts?: ServiceRequestOptions) {
    const categories: ServiceCategory[] = serviceCategories.map((category) => ({
      ...category,
      services: category.services.map((service) => ({
        ...service,
        state: assuranceLevel === 'none' ? service.state : service.stepUpRequired ? 'STEP_UP' : 'READY',
      })),
    }));
    return simulate(categories, opts);
  },

  getTimeline(opts?: ServiceRequestOptions) {
    return simulate(timelineFull, opts);
  },

  getTimelinePreview(opts?: ServiceRequestOptions) {
    return simulate(timelineFull.slice(0, 3), opts);
  },

  getAttributes(opts?: ServiceRequestOptions) {
    return simulate(attributes, opts);
  },

  getDepartmentRecords(opts?: ServiceRequestOptions) {
    return simulate(departmentRecords, opts);
  },

  getConsents(opts?: ServiceRequestOptions) {
    return simulate(consents, opts);
  },

  /** Revokes the micro app's consent grant at the identity server. In the
   * real integration this calls WSO2 Identity Server's consent-management
   * API; the app is expected to re-prompt consent next time it is opened. */
  revokeConsent(id: string, opts?: ServiceRequestOptions) {
    consents = consents.filter((c) => c.id !== id);
    return simulate(consents, opts);
  },

  getWalletDocuments(opts?: ServiceRequestOptions) {
    return simulate(walletDocuments, opts);
  },
};
