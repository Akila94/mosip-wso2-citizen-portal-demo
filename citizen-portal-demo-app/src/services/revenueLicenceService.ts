/**
 * Micro app B (Vehicle Revenue Licence) — Wireframe D's SSO payoff, now real.
 *
 * Every call here goes to `/bff/revenue-licence/api/…` and is answered with
 * **Application B's own** access token. The vehicles come from the Transport
 * registry, looked up server-side against the citizen's verified subject, so
 * the wireframe's central claim — that the citizen never types a plate number
 * and never signs in a second time — is now literally true rather than staged.
 */
import type { CitizenProfile, VehicleRecord, VerifiedIdentitySummary } from './types';
import { apiGet, apiPost } from './http';
import { CLIENTS } from '../config/clients';
import { toVerifiedIdentity } from './verifiedIdentity';

const VRL = 'revenue-licence' as const;

export const revenueLicenceService = {
  getVehicles(): Promise<VehicleRecord[]> {
    return apiGet<VehicleRecord[]>(VRL, '/vehicles');
  },

  /**
   * The narrower verified-identity panel this app shows.
   *
   * Nothing here trims the record down for presentation: this app's token
   * simply carries fewer scopes than the Driving Licence Service's, so
   * `gov-services-api` releases fewer fields to it. The panel is narrower
   * because the grant is narrower — which is the wireframe's point, now
   * enforced by the resource server rather than by a hardcoded literal.
   */
  async getVerifiedIdentity(): Promise<VerifiedIdentitySummary> {
    const profile = await apiGet<CitizenProfile>(VRL, '/identity');
    return toVerifiedIdentity(profile, CLIENTS[VRL].clientName);
  },

  /** Renews one vehicle's licence. CSRF-protected; see `http.ts`'s `apiPost`. */
  renewLicence(vehicleId: string): Promise<{ receiptRef: string }> {
    return apiPost<{ receiptRef: string }>(VRL, `/vehicles/${encodeURIComponent(vehicleId)}/renew`);
  },
};
