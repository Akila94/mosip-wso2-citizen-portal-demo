import React, { useEffect, useState } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate, useParams } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { LandingScreen } from './screens/LandingScreen';
import { TimelineScreen } from './screens/TimelineScreen';
import { ProfileConsentsScreen } from './screens/ProfileConsentsScreen';
import { DocumentsScreen } from './screens/DocumentsScreen';
import { ServiceDetailScreen } from './screens/ServiceDetailScreen';
import { IdentityLoginScreen } from './screens/IdentityLoginScreen';
import { FederatedIdpScreen } from './screens/FederatedIdpScreen';
import { ConsentScreen } from './screens/ConsentScreen';
import { StepUpAuthScreen } from './screens/StepUpAuthScreen';
import { SessionExpiredDemoScreen } from './screens/SessionExpiredDemoScreen';
import { ApplicationStep1Screen } from './screens/ApplicationStep1Screen';
import { ApplicationStep2Screen } from './screens/ApplicationStep2Screen';
import { ApplicationStep3Screen } from './screens/ApplicationStep3Screen';
import { ApplicationStep4Screen } from './screens/ApplicationStep4Screen';
import { ApplicationConfirmationScreen } from './screens/ApplicationConfirmationScreen';
import { ApplicationErrorScreen } from './screens/ApplicationErrorScreen';
import { VehicleRevenueLicenceScreen } from './screens/VehicleRevenueLicenceScreen';
import { AdminConsoleScreen } from './screens/AdminConsoleScreen';
import { DeferredAuthStep1Screen } from './screens/DeferredAuthStep1Screen';
import { DemoNav } from './components/identity/DemoNav';
import { AiAssistantWidget } from './components/assistant/AiAssistantWidget';
import { useLicenceApplicationWizard } from './hooks/useLicenceApplicationWizard';

export type Screen =
  | 'landing'
  | 'timeline'
  | 'profile'
  | 'documents'
  | 'serviceDetail'
  | 'identityLogin'
  | 'federatedIdp'
  | 'consent'
  | 'stepUp'
  | 'sessionExpiredDemo'
  | 'appStep1'
  | 'appStep2'
  | 'appStep3'
  | 'appStep4'
  | 'applicationConfirmation'
  | 'applicationError'
  | 'vehicleRevenueLicence'
  | 'adminConsole'
  | 'deferredAuthStep1';

/** Single source of truth for the Screen -> URL mapping (closes README
 * inconsistency #3 — one naming scheme instead of a 20-value flat switch).
 * `serviceDetail` is the only entry that needs data beyond the screen name
 * itself, so it reads the caller-supplied service id out of `ctx`. */
function screenToPath(screen: Screen, ctx: { serviceId: string }): string {
  switch (screen) {
    case 'landing':
      return '/';
    case 'timeline':
      return '/timeline';
    case 'profile':
      return '/profile';
    case 'documents':
      return '/documents';
    case 'serviceDetail':
      return `/services/${ctx.serviceId}`;
    case 'identityLogin':
      return '/wireframes/identity-login';
    case 'federatedIdp':
      return '/wireframes/federated-idp';
    case 'consent':
      return '/wireframes/consent';
    case 'stepUp':
      return '/wireframes/step-up';
    case 'sessionExpiredDemo':
      return '/wireframes/session-expired';
    case 'deferredAuthStep1':
      return '/wireframes/deferred-auth';
    case 'adminConsole':
      return '/wireframes/admin-console';
    case 'appStep1':
      return '/apps/driving-licence/step/1';
    case 'appStep2':
      return '/apps/driving-licence/step/2';
    case 'appStep3':
      return '/apps/driving-licence/step/3';
    case 'appStep4':
      return '/apps/driving-licence/step/4';
    case 'applicationConfirmation':
      return '/apps/driving-licence/confirmation';
    case 'applicationError':
      return '/apps/driving-licence/error';
    case 'vehicleRevenueLicence':
      return '/apps/revenue-licence';
  }
}

const DEMO_NAV_VISIBLE_PATHS = new Set([
  '/wireframes/identity-login',
  '/wireframes/federated-idp',
  '/wireframes/consent',
  '/wireframes/step-up',
  '/wireframes/session-expired',
  '/apps/revenue-licence',
  '/wireframes/admin-console',
  '/wireframes/deferred-auth',
]);

/** Reads the :serviceId route param and keeps Shell's activeServiceId state
 * (still the single source of truth every other screen in the service-entry
 * flow reads) in sync with the URL, so a direct link to
 * /services/:serviceId works exactly like clicking through from Landing. */
function ServiceDetailRoute({
  activeServiceId,
  onServiceIdChange,
  onNavigate,
  onStartApplication,
  onHeaderSignIn,
}: {
  activeServiceId: string;
  onServiceIdChange: (serviceId: string) => void;
  onNavigate: (screen: Screen) => void;
  onStartApplication: (serviceId: string) => void;
  onHeaderSignIn: () => void;
}) {
  const { serviceId } = useParams<{ serviceId: string }>();

  useEffect(() => {
    if (serviceId && serviceId !== activeServiceId) onServiceIdChange(serviceId);
  }, [serviceId, activeServiceId, onServiceIdChange]);

  return (
    <ServiceDetailScreen
      serviceId={serviceId ?? activeServiceId}
      onNavigate={onNavigate}
      onStartApplication={onStartApplication}
      onHeaderSignIn={onHeaderSignIn}
    />
  );
}

/** No state-switch dependency any more — every screen is a real route,
 * reachable by URL. Screens keep their existing `onNavigate(screen)` prop
 * contract unchanged; `goTo` is the adapter translating a Screen value into
 * a `navigate()` call via `screenToPath`. */
function Shell() {
  const { signIn, signInWithWallet, raiseAssurance, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [activeServiceId, setActiveServiceId] = useState('svc-dl');
  const [stepUpReturn, setStepUpReturn] = useState<Screen>('landing');
  const [loginSuccessTarget, setLoginSuccessTarget] = useState<Screen>('consent');
  const [loginCancelTarget, setLoginCancelTarget] = useState<Screen>('serviceDetail');
  const wizard = useLicenceApplicationWizard();

  function goTo(next: Screen) {
    navigate(screenToPath(next, { serviceId: activeServiceId }));
  }

  function handleSelectService(serviceId: string) {
    setActiveServiceId(serviceId);
    navigate(`/services/${serviceId}`);
  }

  function handleStartApplication(serviceId: string) {
    setActiveServiceId(serviceId);
    setLoginSuccessTarget('consent');
    setLoginCancelTarget('serviceDetail');
    // Already signed in (SSO): no need to log into this micro app again —
    // only its consent grant is still per-app, per the wireframe's "one
    // sign-in, many consent grants" model.
    if (isAuthenticated) {
      goTo('consent');
    } else {
      goTo('identityLogin');
    }
  }

  /** Header's generic "Sign in" — not tied to any specific service, so
   * success and cancel both return to the main (landing) page, now
   * authenticated, per product decision. */
  function handleHeaderSignIn() {
    setLoginSuccessTarget('landing');
    setLoginCancelTarget('landing');
    goTo('identityLogin');
  }

  function handleFederatedSelect(idpId: string, externalHop: boolean) {
    if (idpId === 'wallet') {
      // The one real path: full-page redirect through the BFF to ThunderID.
      // loginSuccessTarget is plain React state, wiped by the navigation
      // away from the SPA, so it's resolved to a URL path now and carried
      // through as returnTo instead.
      signInWithWallet(screenToPath(loginSuccessTarget, { serviceId: activeServiceId }));
      return;
    }
    if (externalHop) {
      goTo('federatedIdp');
    } else {
      signIn('basic');
      goTo(loginSuccessTarget);
    }
  }

  function enterStepUp(returnTo: Screen) {
    setStepUpReturn(returnTo);
    goTo('stepUp');
  }

  function handleStep3Continue() {
    goTo(wizard.triggersMedicalReview ? 'applicationError' : 'appStep4');
  }

  return (
    <>
      <Routes>
        <Route path="/" element={<LandingScreen onNavigate={goTo} onSelectService={handleSelectService} onHeaderSignIn={handleHeaderSignIn} />} />
        <Route path="/timeline" element={<TimelineScreen onNavigate={goTo} />} />
        <Route path="/profile" element={<ProfileConsentsScreen onNavigate={goTo} />} />
        <Route path="/documents" element={<DocumentsScreen onNavigate={goTo} />} />
        <Route
          path="/services/:serviceId"
          element={
            <ServiceDetailRoute
              activeServiceId={activeServiceId}
              onServiceIdChange={setActiveServiceId}
              onNavigate={goTo}
              onStartApplication={handleStartApplication}
              onHeaderSignIn={handleHeaderSignIn}
            />
          }
        />
        <Route
          path="/wireframes/identity-login"
          element={
            <IdentityLoginScreen
              serviceId={activeServiceId}
              onLocalSignIn={() => {
                signIn('basic');
                goTo(loginSuccessTarget);
              }}
              onFederatedSelect={handleFederatedSelect}
              onCancel={() => goTo(loginCancelTarget)}
            />
          }
        />
        <Route
          path="/wireframes/federated-idp"
          element={
            <FederatedIdpScreen
              onVerified={() => {
                signIn('substantial');
                goTo(loginSuccessTarget);
              }}
            />
          }
        />
        <Route
          path="/wireframes/consent"
          element={<ConsentScreen serviceId={activeServiceId} onAllow={() => goTo('appStep1')} onDeny={() => goTo('serviceDetail')} />}
        />
        <Route
          path="/wireframes/step-up"
          element={
            <StepUpAuthScreen
              onConfirmed={() => {
                raiseAssurance('substantial');
                goTo(stepUpReturn);
              }}
              onUseAnotherMethod={() => goTo('stepUp')}
            />
          }
        />
        <Route
          path="/wireframes/session-expired"
          element={<SessionExpiredDemoScreen onStartNewSession={() => goTo('identityLogin')} onBackToPortal={() => goTo('landing')} />}
        />
        <Route path="/wireframes/deferred-auth" element={<DeferredAuthStep1Screen onNavigate={goTo} />} />
        <Route path="/wireframes/admin-console" element={<AdminConsoleScreen />} />
        <Route
          path="/apps/driving-licence/step/1"
          element={<ApplicationStep1Screen wizard={wizard} onBack={() => goTo('serviceDetail')} onContinue={() => goTo('appStep2')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />}
        />
        <Route
          path="/apps/driving-licence/step/2"
          element={<ApplicationStep2Screen wizard={wizard} onBack={() => goTo('appStep1')} onContinue={() => goTo('appStep3')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />}
        />
        <Route
          path="/apps/driving-licence/step/3"
          element={<ApplicationStep3Screen wizard={wizard} onBack={() => goTo('appStep2')} onContinue={handleStep3Continue} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />}
        />
        <Route
          path="/apps/driving-licence/step/4"
          element={<ApplicationStep4Screen wizard={wizard} onBack={() => goTo('appStep3')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} onSubmit={() => enterStepUp('applicationConfirmation')} />}
        />
        <Route path="/apps/driving-licence/confirmation" element={<ApplicationConfirmationScreen onNavigate={goTo} />} />
        <Route path="/apps/driving-licence/error" element={<ApplicationErrorScreen onNavigate={goTo} />} />
        <Route path="/apps/revenue-licence" element={<VehicleRevenueLicenceScreen onSaveExit={() => goTo('landing')} />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      {DEMO_NAV_VISIBLE_PATHS.has(location.pathname) && <DemoNav onNavigate={goTo} />}
      <AiAssistantWidget onStartRenewal={() => goTo('appStep1')} />
    </>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Shell />
      </AuthProvider>
    </BrowserRouter>
  );
}
