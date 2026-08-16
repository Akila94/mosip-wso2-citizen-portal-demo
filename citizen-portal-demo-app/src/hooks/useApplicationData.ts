import { useAsync } from './useAsync';
import { applicationService } from '../services/applicationService';
import type { ApplicationConfig, ApplicationErrorInfo } from '../services/types';

export function useApplicationConfig() {
  return useAsync<ApplicationConfig>(() => applicationService.getApplicationConfig(), []);
}

export function useMedicalReviewError() {
  return useAsync<ApplicationErrorInfo>(() => applicationService.getMedicalReviewError(), []);
}
