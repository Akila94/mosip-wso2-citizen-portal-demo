/**
 * Builds the read-only "verified identity" panel content from the real
 * government registry record, replacing the hardcoded `verifiedIdentity` and
 * `NARROW_IDENTITY` literals the two micro apps each used to carry (this app's
 * README inconsistency #1).
 *
 * The interesting property is that both micro apps call the *same* registry
 * endpoint and get *different* answers: `gov-services-api` projects the record
 * by the scopes the calling app's token actually carries. The Driving Licence
 * Service holds `address`, so it sees NIC and address; the Vehicle Revenue
 * Licence does not, so it sees neither. The panel therefore renders exactly
 * what was released — never a placeholder standing in for a field this app was
 * not entitled to.
 */
import type { CitizenProfile, VerifiedIdentitySummary } from './types';

/**
 * Where these facts come from, stated accurately.
 *
 * Deliberately *not* "verified — national digital ID": the citizen may have
 * signed in with a username and password rather than eSignet, and these
 * attributes are the government registry's record either way, looked up
 * against the verified subject. Naming the registry is true in both cases.
 */
const REGISTRY_BADGE = 'VERIFIED — GOVERNMENT REGISTRY';

/** Shown when the registry released no name to this application. */
const UNNAMED = 'Name not released to this service';

export function toVerifiedIdentity(profile: CitizenProfile, clientName: string): VerifiedIdentitySummary {
  const attributes = [];

  if (profile.nic) attributes.push({ label: 'NIC number', value: profile.nic });
  if (profile.birthdate) attributes.push({ label: 'Date of birth', value: profile.birthdate });
  if (profile.address) attributes.push({ label: 'Address', value: profile.address });

  // Always last, and always present: the point of the panel is that the
  // citizen can see which application received these facts.
  attributes.push({ label: 'Released to', value: clientName });

  return {
    name: profile.name?.trim() || UNNAMED,
    badgeLabel: REGISTRY_BADGE,
    attributes,
  };
}
