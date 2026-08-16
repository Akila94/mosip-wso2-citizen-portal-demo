import { useAsync } from './useAsync';
import { adminConsoleService } from '../services/adminConsoleService';
import type { AdminNavItem, ScopeOption, ServiceDraft } from '../services/types';

export function useAdminNav() {
  return useAsync<AdminNavItem[]>(() => adminConsoleService.getNavItems(), []);
}

export function useScopeCatalogue() {
  return useAsync<ScopeOption[]>(() => adminConsoleService.getScopeCatalogue(), []);
}

export function useServiceDraft() {
  return useAsync<ServiceDraft>(() => adminConsoleService.getDraft(), []);
}
