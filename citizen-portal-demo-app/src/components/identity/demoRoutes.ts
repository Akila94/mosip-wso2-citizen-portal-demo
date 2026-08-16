import type { Screen } from '../../App';

export const DEMO_ROUTES: { screen: Screen; label: string }[] = [
  { screen: 'landing', label: 'Portal home' },
  { screen: 'stepUp', label: 'Step-up MFA (frame 11)' },
  { screen: 'sessionExpiredDemo', label: 'Session expired (frame 12)' },
  { screen: 'vehicleRevenueLicence', label: 'SSO payoff — Revenue Licence (frame 19)' },
  { screen: 'adminConsole', label: 'Admin console (frame 20, optional)' },
  { screen: 'deferredAuthStep1', label: 'Deferred auth variant (frame 23, comparison only)' },
];
