import { useAsync } from './useAsync';
import { portalService } from '../services/portalService';
import { useAuth } from '../context/AuthContext';
import type {
  LifeEvent,
  ServiceCategory,
  TimelineItem,
  AttributeRecord,
  DepartmentRecord,
  ConsentGrant,
  WalletDocument,
} from '../services/types';

export function useLandingData() {
  const { assuranceLevel } = useAuth();
  const lifeEvents = useAsync<LifeEvent[]>(() => portalService.getLifeEvents(), []);
  const categories = useAsync<ServiceCategory[]>(() => portalService.getServiceCatalogue(assuranceLevel), [assuranceLevel]);
  const timelinePreview = useAsync<TimelineItem[]>(() => portalService.getTimelinePreview(), []);
  return { lifeEvents, categories, timelinePreview };
}

export function useTimelineData() {
  return useAsync<TimelineItem[]>(() => portalService.getTimeline(), []);
}

export function useProfileData() {
  const attributes = useAsync<AttributeRecord[]>(() => portalService.getAttributes(), []);
  const records = useAsync<DepartmentRecord[]>(() => portalService.getDepartmentRecords(), []);
  const consents = useAsync<ConsentGrant[]>(() => portalService.getConsents(), []);
  return { attributes, records, consents };
}

export function useDocumentsData() {
  return useAsync<WalletDocument[]>(() => portalService.getWalletDocuments(), []);
}
