import type { SessionInspectorData, ServiceRequestOptions } from './types';

const DEFAULT_DELAY_MS = 400;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Could not reach the session inspector.'));
      else resolve(payload);
    }, delayMs);
  });
}

const baseSessionRows = [
  { label: 'subject', value: 'a7f3•••-4471' },
  { label: 'idp', value: 'National Digital ID' },
  { label: 'acr', value: 'substantial' },
  { label: 'amr', value: 'eid, otp' },
  { label: 'session started', value: '14:02:11' },
  { label: 'remaining lifetime', value: '51 min' },
];

/**
 * Demo-only tool (per the wireframe: "toggled from the footer, hidden for
 * non-technical rooms"). Same underlying session, different claim sets per
 * client — scope, not session, decides what each micro app sees. A real
 * implementation would read this from WSO2 Identity Server's introspection
 * or userinfo endpoint for the current client.
 */
const byClient: Record<string, SessionInspectorData> = {
  'driving-licence': {
    clientLabel: 'Driving Licence Service',
    sessionRows: [...baseSessionRows, { label: 'clients in session', value: '2 — Driving Licence, Revenue Licence' }],
    claims: {
      sub: 'a7f3•••-4471',
      name: 'John Doe',
      nic: '19•• ••• •••• 4471',
      address: '14 Lake Road, Marolia City',
      acr: 'substantial',
      amr: ['eid', 'otp'],
      aud: 'driving-licence',
      iss: 'https://identity.marolia.gov',
    },
    comparisonNote: 'This app also receives address and photograph — a wider scope than Revenue Licence needs.',
  },
  'vehicle-revenue-licence': {
    clientLabel: 'Vehicle Revenue Licence',
    sessionRows: [...baseSessionRows, { label: 'clients in session', value: '2 — Driving Licence, Revenue Licence' }],
    claims: {
      sub: 'a7f3•••-4471',
      name: 'John Doe',
      nic: '19•• ••• •••• 4471',
      acr: 'substantial',
      amr: ['eid', 'otp'],
      aud: 'vehicle-revenue-licence',
      iss: 'https://identity.marolia.gov',
    },
    comparisonNote: 'Note the narrower claim set than the licence app received — no address, no photograph.',
  },
};

export const sessionInspectorService = {
  getSessionData(clientId: string, opts?: ServiceRequestOptions) {
    return simulate(byClient[clientId] ?? byClient['vehicle-revenue-licence'], opts);
  },
};
