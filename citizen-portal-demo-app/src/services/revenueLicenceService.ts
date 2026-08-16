import type { VehicleRecord, ServiceRequestOptions } from './types';

const DEFAULT_DELAY_MS = 500;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Could not reach the revenue licence service. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

let vehicles: VehicleRecord[] = [
  {
    id: 'veh-cab4471',
    plate: 'CAB-4471',
    description: 'Motor car · 1,300cc · petrol',
    dueDate: 'DUE 30 SEP',
    fee: '$15',
    owner: 'J. Doe (you)',
    province: 'Marolia Central',
    registryChecks: [
      { label: 'Insurance check', status: 'Valid to 12 Nov 2026 · confirmed with your insurer automatically' },
      { label: 'Emissions test', status: 'Passed 04 Jun 2026 · valid' },
    ],
  },
];

/**
 * Micro app B's own data domain (registered vehicles + renewal), pulled
 * from the Transport registry against the citizen's verified NIC — the
 * citizen never types a plate number, per the wireframe's SSO-payoff point.
 */
export const revenueLicenceService = {
  getVehicles(opts?: ServiceRequestOptions) {
    return simulate(vehicles, opts);
  },

  renewLicence(vehicleId: string, opts?: ServiceRequestOptions) {
    vehicles = vehicles.map((v) => (v.id === vehicleId ? { ...v, dueDate: 'RENEWED · valid to 30 Sep 2027' } : v));
    return simulate({ receiptRef: `PAY-VRL-${Date.now().toString().slice(-6)}` }, opts);
  },
};
