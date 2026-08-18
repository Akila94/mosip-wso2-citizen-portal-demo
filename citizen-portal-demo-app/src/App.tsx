import React, { useEffect, useState } from 'react';
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation, useNavigate, useParams } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { CLIENTS, clientForService, type AppKey } from './config/clients';
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

/** Single source of truth for the Screen -> URL mapping. */
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

/** Neutral holding screen shown while a session is being established. */
function AuthSplash({ message }: { message: string }) {
  return (
    <div role="status" aria-live="polite" style={{ minHeight: '60vh', display: 'flex', alignItems: 'center', justifyContent: 'center', font: '400 13px var(--font-mono)', color: 'var(--text-secondary)' }}>
      {message}
    </div>
  );
}

/**
 * Guards a route tree, sending an unauthenticated citizen to WSO2 IS and
 * bringing them back to where they were aiming.
 *
 * This is what makes "cold entry" work: opening a micro-app URL directly, with
 * no portal login, produces a real `/authorize` round trip. If an IS session
 * already exists — because the citizen signed in at the portal, or in the
 * other micro app — IS answers it without prompting, which is the SSO the demo
 * is built to show.
 *
 * The splash matters: without it every guarded page would flash a signed-out
 * state during the session bootstrap, before the redirect it is about to make.
 */
function RequireAuth() {
  const { isAuthenticated, isLoading, signIn } = useAuth();
  const location = useLocation();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      signIn(location.pathname + location.search);
    }
  }, [isLoading, isAuthenticated, signIn, location.pathname, location.search]);

  if (isLoading) return <AuthSplash message="Checking your session…" />;
  if (!isAuthenticated) return <AuthSplash message="Redirecting you to sign in…" />;
  return <Outlet />;
}

/**
 * A micro app's route tree: its own OIDC client session, then the guard.
 *
 * Nesting a second AuthProvider here is the point, not an accident — each
 * micro app is a separate registered application with its own token, its own
 * audience and its own released claims, so it reads its own BFF namespace.
 * The portal's provider still wraps this one, which is why the session
 * inspector can truthfully report several clients sharing one IdP session.
 */
function MicroAppLayout({ appKey }: { appKey: AppKey }) {
  return (
    <AuthProvider appKey={appKey}>
      <RequireAuth />
    </AuthProvider>
  );
}

/** Keeps Shell's activeServiceId in sync with the :serviceId in the URL. */
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

function Shell() {
  const { signIn } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [activeServiceId, setActiveServiceId] = useState('svc-dl');
  const [stepUpReturn, setStepUpReturn] = useState<Screen>('landing');
  const [loginCancelTarget, setLoginCancelTarget] = useState<Screen>('serviceDetail');
  const wizard = useLicenceApplicationWizard();

  function goTo(next: Screen) {
    navigate(screenToPath(next, { serviceId: activeServiceId }));
  }

  function handleSelectService(serviceId: string) {
    setActiveServiceId(serviceId);
    navigate(`/services/${serviceId}`);
  }

  /**
   * Enters the micro app that implements this service.
   *
   * There is no consent stop here any more: WSO2 IS owns consent, and it
   * shows its own consent page during the `/authorize` round trip that
   * `RequireAuth` triggers on arrival. A service with no micro app (every
   * STUB entry in the catalogue) leaves the citizen where they are.
   */
  function handleStartApplication(serviceId: string) {
    setActiveServiceId(serviceId);
    setLoginCancelTarget('serviceDetail');
    const client = clientForService(serviceId);
    if (!client) return;
    navigate(client.appKey === 'driving-licence' ? '/apps/driving-licence/step/1' : client.routeBase);
  }

  /** Header's "Sign in" — the real thing: a redirect to WSO2 IS. */
  function handleHeaderSignIn() {
    signIn(location.pathname + location.search);
  }

  /**
   * Wireframe-only provider picker. It navigates the reference screens and
   * deliberately does not authenticate — see IdentityLoginScreen's own note.
   */
  function handleFederatedSelect(_idpId: string, externalHop: boolean) {
    goTo(externalHop ? 'federatedIdp' : 'landing');
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
        {/* Public portal routes: the landing page and service detail are
            deliberately reachable signed out — that is the wireframe's
            "sign in only when you're ready to apply" contract. */}
        <Route path="/" element={<LandingScreen onNavigate={goTo} onSelectService={handleSelectService} onHeaderSignIn={handleHeaderSignIn} />} />
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

        {/* Portal routes that show the citizen's own data, so they need the
            portal session that already wraps this component. */}
        <Route element={<RequireAuth />}>
          <Route path="/timeline" element={<TimelineScreen onNavigate={goTo} />} />
          <Route path="/profile" element={<ProfileConsentsScreen onNavigate={goTo} />} />
          <Route path="/documents" element={<DocumentsScreen onNavigate={goTo} />} />
        </Route>

        {/* Application A — its own OIDC client. */}
        <Route element={<MicroAppLayout appKey="driving-licence" />}>
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
        </Route>

        {/* Application B — a different OIDC client, a different audience. */}
        <Route element={<MicroAppLayout appKey="revenue-licence" />}>
          <Route path="/apps/revenue-licence" element={<VehicleRevenueLicenceScreen onSaveExit={() => goTo('landing')} />} />
        </Route>

        {/* Reference wireframes. WSO2 IS now owns login, provider choice and
            consent; these survive to narrate what it does and authenticate
            nothing. */}
        <Route
          path="/wireframes/identity-login"
          element={
            <IdentityLoginScreen
              serviceId={activeServiceId}
              onLocalSignIn={() => goTo('landing')}
              onFederatedSelect={handleFederatedSelect}
              onCancel={() => goTo(loginCancelTarget)}
            />
          }
        />
        <Route path="/wireframes/federated-idp" element={<FederatedIdpScreen onVerified={() => goTo('landing')} />} />
        <Route path="/wireframes/consent" element={<ConsentScreen serviceId={activeServiceId} onAllow={() => goTo('appStep1')} onDeny={() => goTo('serviceDetail')} />} />
        <Route path="/wireframes/step-up" element={<StepUpAuthScreen onConfirmed={() => goTo(stepUpReturn)} onUseAnotherMethod={() => goTo('stepUp')} />} />
        <Route path="/wireframes/session-expired" element={<SessionExpiredDemoScreen onStartNewSession={() => signIn('/')} onBackToPortal={() => goTo('landing')} />} />
        <Route path="/wireframes/deferred-auth" element={<DeferredAuthStep1Screen onNavigate={goTo} />} />
        <Route path="/wireframes/admin-console" element={<AdminConsoleScreen />} />

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
      {/* The portal's own session wraps everything, including the globally
          mounted assistant widget. Micro apps nest their own provider inside
          this one. */}
      <AuthProvider appKey={CLIENTS.portal.appKey}>
        <Shell />
      </AuthProvider>
    </BrowserRouter>
  );
}
