import type { AdminNavItem, RegistrationFieldDef, ScopeOption, ServiceDraft, ServiceRequestOptions } from './types';

const DEFAULT_DELAY_MS = 500;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Could not reach the platform console.'));
      else resolve(payload);
    }, delayMs);
  });
}

const navItems: AdminNavItem[] = [
  { id: 'applications', label: 'Applications' },
  { id: 'idps', label: 'Identity providers' },
  { id: 'scopes', label: 'Scopes & claims' },
  { id: 'consent-policies', label: 'Consent policies' },
  { id: 'catalogue', label: 'Service catalogue' },
];

const registrationFields: RegistrationFieldDef[] = [
  { id: 'serviceName', label: 'Service name (citizen-facing)', value: 'Birth Certificate Service', help: 'Appears on the catalogue card and every consent screen.' },
  { id: 'owningAgency', label: 'Owning agency', value: 'Registrar General', help: 'Shown to citizens as "operated by".' },
  { id: 'redirectUris', label: 'Redirect URIs', value: 'https://birth.marolia.gov/callback', help: 'One per line. Exact match enforced.' },
  { id: 'postLogoutRedirect', label: 'Post-logout redirect', value: 'https://marolia.gov/signed-out', help: 'Used by single logout across all micro apps.' },
];

const scopeCatalogue: ScopeOption[] = [
  { id: 'openid', label: 'openid', description: 'required for any OIDC client', requiredByDefault: true },
  { id: 'profile', label: 'profile', description: 'full name', requiredByDefault: true },
  { id: 'nic', label: 'nic', description: 'national identity number', requiredByDefault: true },
  { id: 'address', label: 'address', description: 'registered address', requiredByDefault: false },
  { id: 'civil_registry.read', label: 'civil_registry.read', description: "read the citizen's birth record", requiredByDefault: true },
];

let draft: ServiceDraft = {
  serviceName: registrationFields[0].value,
  owningAgency: registrationFields[1].value,
  redirectUris: registrationFields[2].value,
  postLogoutRedirect: registrationFields[3].value,
  selectedScopeIds: scopeCatalogue.filter((s) => s.requiredByDefault).map((s) => s.id),
  assuranceRequirement: 'substantial',
  consentPreview: 'Birth Certificate Service, operated by the Registrar General, is requesting your full name and NIC number to locate your birth record. Retained for the life of the request plus 6 years.',
  status: 'draft',
};

export const adminConsoleService = {
  getNavItems(opts?: ServiceRequestOptions) {
    return simulate(navItems, opts);
  },

  getRegistrationFields(opts?: ServiceRequestOptions) {
    return simulate(registrationFields, opts);
  },

  getScopeCatalogue(opts?: ServiceRequestOptions) {
    return simulate(scopeCatalogue, opts);
  },

  getDraft(opts?: ServiceRequestOptions) {
    return simulate(draft, opts);
  },

  saveDraft(patch: Partial<ServiceDraft>, opts?: ServiceRequestOptions) {
    draft = { ...draft, ...patch, status: 'draft' };
    return simulate(draft, opts);
  },

  submitForReview(patch: Partial<ServiceDraft>, opts?: ServiceRequestOptions) {
    draft = { ...draft, ...patch, status: 'submitted' };
    return simulate(draft, opts);
  },
};
