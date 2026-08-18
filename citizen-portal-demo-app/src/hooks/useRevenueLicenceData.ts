import { useAsync } from './useAsync';
import { revenueLicenceService } from '../services/revenueLicenceService';
import type { VehicleRecord, VerifiedIdentitySummary } from '../services/types';

export function useVehicles() {
  return useAsync<VehicleRecord[]>(() => revenueLicenceService.getVehicles(), []);
}

/**
 * This micro app's verified-identity panel, built from the government registry
 * record projected by *this* app's scopes — deliberately narrower than the
 * Driving Licence Service's, because its token carries fewer.
 */
export function useRevenueLicenceIdentity() {
  return useAsync<VerifiedIdentitySummary>(() => revenueLicenceService.getVerifiedIdentity(), []);
}
