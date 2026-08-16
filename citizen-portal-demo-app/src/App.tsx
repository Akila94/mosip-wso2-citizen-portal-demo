import React, { useState } from 'react';
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

const DEMO_NAV_VISIBLE_SCREENS: Screen[] = ['identityLogin', 'federatedIdp', 'consent', 'stepUp', 'sessionExpiredDemo', 'vehicleRevenueLicence', 'adminConsole', 'deferredAuthStep1'];

/** No router dependency, by product decision — a plain state switch stands
 * in for real routes. Swap for react-router (or a host shell's router)
 * without touching screens: each already takes (onNavigate) as a prop. */
function Shell() {
  const { signIn, raiseAssurance, isAuthenticated } = useAuth();
  const [screen, setScreen] = useState<Screen>('landing');
  const [activeServiceId, setActiveServiceId] = useState('svc-dl');
  const [stepUpReturn, setStepUpReturn] = useState<Screen>('landing');
  const [loginSuccessTarget, setLoginSuccessTarget] = useState<Screen>('consent');
  const [loginCancelTarget, setLoginCancelTarget] = useState<Screen>('serviceDetail');
  const wizard = useLicenceApplicationWizard();

  function goTo(next: Screen) {
    setScreen(next);
  }

  function handleSelectService(serviceId: string) {
    setActiveServiceId(serviceId);
    goTo('serviceDetail');
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

  let body: React.ReactNode;
  switch (screen) {
    case 'timeline':
      body = <TimelineScreen onNavigate={goTo} />;
      break;
    case 'profile':
      body = <ProfileConsentsScreen onNavigate={goTo} />;
      break;
    case 'documents':
      body = <DocumentsScreen onNavigate={goTo} />;
      break;
    case 'serviceDetail':
      body = <ServiceDetailScreen serviceId={activeServiceId} onNavigate={goTo} onStartApplication={handleStartApplication} onHeaderSignIn={handleHeaderSignIn} />;
      break;
    case 'identityLogin':
      body = (
        <IdentityLoginScreen
          serviceId={activeServiceId}
          onLocalSignIn={() => {
            signIn('basic');
            goTo(loginSuccessTarget);
          }}
          onFederatedSelect={handleFederatedSelect}
          onCancel={() => goTo(loginCancelTarget)}
        />
      );
      break;
    case 'federatedIdp':
      body = (
        <FederatedIdpScreen
          onVerified={() => {
            signIn('substantial');
            goTo(loginSuccessTarget);
          }}
        />
      );
      break;
    case 'consent':
      body = <ConsentScreen serviceId={activeServiceId} onAllow={() => goTo('appStep1')} onDeny={() => goTo('serviceDetail')} />;
      break;
    case 'stepUp':
      body = (
        <StepUpAuthScreen
          onConfirmed={() => {
            raiseAssurance('substantial');
            goTo(stepUpReturn);
          }}
          onUseAnotherMethod={() => goTo('stepUp')}
        />
      );
      break;
    case 'sessionExpiredDemo':
      body = <SessionExpiredDemoScreen onStartNewSession={() => goTo('identityLogin')} onBackToPortal={() => goTo('landing')} />;
      break;
    case 'appStep1':
      body = <ApplicationStep1Screen wizard={wizard} onBack={() => goTo('serviceDetail')} onContinue={() => goTo('appStep2')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />;
      break;
    case 'appStep2':
      body = <ApplicationStep2Screen wizard={wizard} onBack={() => goTo('appStep1')} onContinue={() => goTo('appStep3')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />;
      break;
    case 'appStep3':
      body = <ApplicationStep3Screen wizard={wizard} onBack={() => goTo('appStep2')} onContinue={handleStep3Continue} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} />;
      break;
    case 'appStep4':
      body = <ApplicationStep4Screen wizard={wizard} onBack={() => goTo('appStep3')} onJumpStep={(s) => goTo((`appStep${s}`) as Screen)} onSaveExit={() => goTo('landing')} onSubmit={() => enterStepUp('applicationConfirmation')} />;
      break;
    case 'applicationConfirmation':
      body = <ApplicationConfirmationScreen onNavigate={goTo} />;
      break;
    case 'applicationError':
      body = <ApplicationErrorScreen onNavigate={goTo} />;
      break;
    case 'vehicleRevenueLicence':
      body = <VehicleRevenueLicenceScreen onSaveExit={() => goTo('landing')} />;
      break;
    case 'adminConsole':
      body = <AdminConsoleScreen />;
      break;
    case 'deferredAuthStep1':
      body = <DeferredAuthStep1Screen onNavigate={goTo} />;
      break;
    default:
      body = <LandingScreen onNavigate={goTo} onSelectService={handleSelectService} onHeaderSignIn={handleHeaderSignIn} />;
  }

  return (
    <>
      {body}
      {DEMO_NAV_VISIBLE_SCREENS.includes(screen) && <DemoNav onNavigate={goTo} />}
      <AiAssistantWidget onStartRenewal={() => goTo('appStep1')} />
    </>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}
