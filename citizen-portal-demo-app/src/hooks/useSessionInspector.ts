import { useAsync } from './useAsync';
import { sessionInspectorService } from '../services/sessionInspectorService';
import type { SessionInspectorData } from '../services/types';
import type { AppKey } from '../config/clients';

/**
 * Reads the real session behind one registered application. `enabled` is
 * false while the inspector panel is collapsed, so a demo tool nobody has
 * opened never issues a request.
 */
export function useSessionInspector(appKey: AppKey, enabled: boolean) {
  return useAsync<SessionInspectorData>(
    () => (enabled ? sessionInspectorService.getSessionData(appKey) : Promise.resolve(null as unknown as SessionInspectorData)),
    [appKey, enabled]
  );
}
