import { useAsync } from './useAsync';
import { revenueLicenceService } from '../services/revenueLicenceService';
import type { VehicleRecord } from '../services/types';

export function useVehicles() {
  return useAsync<VehicleRecord[]>(() => revenueLicenceService.getVehicles(), []);
}
