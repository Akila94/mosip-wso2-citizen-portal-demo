import { useAsync } from './useAsync';
import { identityService } from '../services/identityService';
import type { ServiceDetail, IdentityProvider, ConsentAttributeRequest } from '../services/types';

export function useServiceDetail(serviceId: string) {
  return useAsync<ServiceDetail>(() => identityService.getServiceDetail(serviceId), [serviceId]);
}

export function useIdentityProviders() {
  return useAsync<IdentityProvider[]>(() => identityService.getIdentityProviders(), []);
}

export function useConsentAttributes(serviceId: string) {
  return useAsync<ConsentAttributeRequest[]>(() => identityService.getConsentAttributes(serviceId), [serviceId]);
}
