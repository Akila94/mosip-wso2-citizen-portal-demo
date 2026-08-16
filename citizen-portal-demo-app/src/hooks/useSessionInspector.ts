import { useAsync } from './useAsync';
import { sessionInspectorService } from '../services/sessionInspectorService';
import type { SessionInspectorData } from '../services/types';

export function useSessionInspector(clientId: string, enabled: boolean) {
  return useAsync<SessionInspectorData>(() => (enabled ? sessionInspectorService.getSessionData(clientId) : Promise.resolve(null as unknown as SessionInspectorData)), [clientId, enabled]);
}
