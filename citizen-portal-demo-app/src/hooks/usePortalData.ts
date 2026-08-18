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

/** Resolves to null without issuing a request — for data a signed-out visitor has no right to. */
const NOT_LOADED = <T,>() => Promise.resolve(null as unknown as T);

/**
 * The landing page's data, which is the one place in the app that must work
 * both signed in and signed out.
 *
 * The catalogue switches source rather than switching off: signed out it comes
 * from the public, unauthenticated endpoint, and signed in from the
 * session-backed one that promotes cards to READY/STEP_UP. The timeline is
 * personal data, so it is not fetched at all until there is a session — a
 * request that could only ever 401 is not worth making, and its error would
 * surface on a screen that does not even render it.
 *
 * `isLoading` gates both: firing during the session bootstrap would race the
 * answer and fetch the wrong variant.
 */
export function useLandingData() {
  const { assuranceLevel, isAuthenticated, isLoading } = useAuth();

  const lifeEvents = useAsync<LifeEvent[]>(() => portalService.getLifeEvents(), []);

  const categories = useAsync<ServiceCategory[]>(() => {
    if (isLoading) return NOT_LOADED<ServiceCategory[]>();
    return isAuthenticated ? portalService.getServiceCatalogue(assuranceLevel) : portalService.getPublicServiceCatalogue();
  }, [assuranceLevel, isAuthenticated, isLoading]);

  const timelinePreview = useAsync<TimelineItem[]>(
    () => (isAuthenticated ? portalService.getTimelinePreview() : NOT_LOADED<TimelineItem[]>()),
    [isAuthenticated]
  );

  return {
    lifeEvents,
    // Stay "loading" through the session bootstrap as well as the fetch:
    // otherwise the catalogue renders empty for a frame before we even know
    // which of its two sources to ask.
    categories: { ...categories, loading: categories.loading || isLoading },
    timelinePreview,
  };
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
