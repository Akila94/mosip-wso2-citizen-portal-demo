/**
 * The session inspector's data source — now the real session rather than a
 * hand-authored claim set per client.
 *
 * Each app asks its *own* BFF namespace, and the BFF answers from its own
 * server-side session state without calling `gov-services-api` at all. Opening
 * the panel on both micro apps side by side therefore shows the same `sub`,
 * `sid`, `idp`, `acr`, `amr` and `auth_time` — one IdP session — with a
 * different `aud` and a different released claim set, because they are
 * different OIDC clients. That contrast is the proof the demo is making.
 */
import type { SessionInspectorData } from './types';
import { apiGet } from './http';
import type { AppKey } from '../config/clients';

export const sessionInspectorService = {
  getSessionData(appKey: AppKey): Promise<SessionInspectorData> {
    return apiGet<SessionInspectorData>(appKey, '/session-inspector');
  },
};
