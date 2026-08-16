# MaroliaGov Portal — Wireframes A, B, C & D

Production React/TypeScript implementation of all four web wireframes: **A (Portal shell)**, **B (Service detail & authentication)**, **C (Micro app A — the 4-step driving licence application)**, and **D (SSO payoff, platform console, session inspector, AI assistant, and the deferred-auth variant)**. Mobile-app wireframes are a separate, later phase and are not referenced here.

## Run it
```
npm install
npm run dev
```

## Structure
```
src/
  design-system/     copied verbatim from the WSO2 Design System (components, tokens, styles.css, font)
  components/
    layout/           header, footer, language switcher, a11y toolbar, account menu, identity badge
    service/          service card, life-event card, catalogue, effort-contract line
    timeline/         timeline row/table, submitted-applications panel
    profile/          attribute row/table, department record card, consent row
    documents/        wallet document card
    identity/         IS/eID shells, radio option, OTP input, consent row, session-expired modal, demo nav
    application/       micro-app header, stepper, verified/self-asserted panels, declaration row,
                       document upload, footer nav, test-slot grid, review
    revenue/          vehicle card (micro app B)
    session/          session inspector panel + footer toggle
    platform/         admin sidebar nav, scope checklist, assurance-level picker
    assistant/        docked AI assistant widget
    common/           Placeholder (labeled boxes), AsyncState (loading/error/empty)
  context/
    AuthContext.tsx   mock session (isAuthenticated, user, assuranceLevel)
  services/
    types.ts                     shapes mirroring an eventual real API contract
    portalService.ts             Wireframe A mock data
    identityService.ts           Wireframe B mock data (auth/consent)
    applicationService.ts        Wireframe C mock data + live test-slot generator
    revenueLicenceService.ts     Wireframe D — micro app B (vehicles, renewal)
    sessionInspectorService.ts   Wireframe D — session/claims per client
    adminConsoleService.ts       Wireframe D — service-registration console
    assistantService.ts          Wireframe D — canned assistant replies
  hooks/
    useAsync.ts, usePortalData.ts, useIdentityData.ts, useApplicationData.ts,
    useLicenceApplicationWizard.ts, useTestSlotBooking.ts,
    useRevenueLicenceData.ts, useSessionInspector.ts, useAdminConsoleData.ts, useAssistant.ts
  screens/
    LandingScreen.tsx, TimelineScreen.tsx, ProfileConsentsScreen.tsx, DocumentsScreen.tsx
    ServiceDetailScreen.tsx, IdentityLoginScreen.tsx, FederatedIdpScreen.tsx,
    ConsentScreen.tsx, StepUpAuthScreen.tsx, SessionExpiredDemoScreen.tsx,
    ApplicationStep1Screen.tsx .. ApplicationStep4Screen.tsx,
    ApplicationConfirmationScreen.tsx, ApplicationErrorScreen.tsx,
    VehicleRevenueLicenceScreen.tsx, AdminConsoleScreen.tsx, DeferredAuthStep1Screen.tsx
```

## Wireframe D
- **Frame 19 — Vehicle Revenue Licence (`VehicleRevenueLicenceScreen`), never cut.** The SSO payoff: a second micro app, reusing the session established in B/C, showing vehicles pulled from a mock Transport registry (`revenueLicenceService`) with no re-typed plate number, plus two registry-check rows. The verified-identity panel here is deliberately **narrower** (NIC + "released to" only — no address/photograph), matching the wireframe's point that each micro app receives only the scope it requested.
- **Frame 20 — Admin console (`AdminConsoleScreen`), optional.** A different persona (agency admin), so it does not reuse the citizen header/footer. Real interactivity: editable registration fields, a scope checklist (`ScopeChecklist`, DS `Checkbox`), an assurance-level picker, a consent-text preview, and working Save draft / Submit for review actions against `adminConsoleService`.
- **Frames 21/22 — Session inspector & AI assistant, optional.** `SessionInspectorPanel` is a real collapsible panel (toggled from the footer, per the wireframe) showing session rows and ID-token claims that genuinely differ between the driving-licence and revenue-licence clients (`sessionInspectorService`) — wired onto both `ApplicationStep2Screen` (frame 14) and `VehicleRevenueLicenceScreen` (frame 19) so the two can be compared, exactly as the wireframe's annotation asks. `AiAssistantWidget` is a working docked chat (56px launcher, bottom-right, mounted globally in `App.tsx` so it's present on every screen) with canned anonymous/authenticated replies (`assistantService`); its "Start the renewal" action really navigates into `appStep1`.
- **Frame 23 — deferred-authentication variant (`DeferredAuthStep1Screen`), comparison-only.** An alternate Step 1 that works without signing in (typed, unverified DOB; a "provisional" eligibility result; a sign-in prompt framed as a convenience). Per the wireframe's own framing ("compare against frame 13 … this exists so the choice is a decision you made"), it is **not** wired into the canonical A→B→C→D flow — reachable only via the demo nav.

## New deviations for D — flagged
- **`AiAssistantWidget`** — no chat/message-bubble primitive in the DS; built from DS surface/border/radius/spacing tokens.
- **`AssuranceLevelPicker`** (admin console) — no 3-option segmented primitive in the DS; same tokens-based pattern as the existing language switcher and payment-method picker.
- **Session inspector is a demo-only tool** (explicitly so in the wireframe) reading mock per-client claim sets, not a real WSO2 IS introspection/userinfo call.
- **Micro app B's renewal is intentionally thin**, per the wireframe's own note ("kept deliberately thin … adding a form here would bury the payoff") — clicking "Renew" calls `revenueLicenceService.renewLicence` and flips the card to a renewed state inline; there is no receipt/confirmation screen for this path.

## Test-slot booking is now live (Wireframe C follow-up)
Step 4's appointment picker (`TestSlotGrid` + `useTestSlotBooking`) is fully interactive: `applicationService.getTestWeek(weekOffset)` returns a real week of slots (simulated latency), clicking an available slot holds it and starts a real 10-minute countdown (`setInterval`, ticking every second) with the hold auto-releasing at 0, and "next/prev week" re-fetches. Submitting step 4 is now blocked until a slot is held. The static `testWeek`/`heldSlot` fields were removed from `ApplicationConfig` since this data is no longer static.

## Wireframe C — the 4-step application
`ConsentScreen (Allow) → ApplicationStep1Screen → …Step4Screen → StepUpAuthScreen (B) → ApplicationConfirmationScreen`, with `ApplicationStep3Screen` routing to `ApplicationErrorScreen` when the blackouts declaration is "yes" instead of continuing to step 4. The persistent verified-identity panel (`VerifiedIdentityPanel`) and, on step 2, the self-asserted panel (`SelfAssertedPanel`) are shared across all four steps — the trust-model split (verified above, self-asserted below, never merged) is the wireframe's central rule and is enforced structurally rather than per-screen. `MicroAppHeader` is shared between C and D (`appName`/`tagline` props) rather than being duplicated per micro app.

**B → C wiring:**
- Consent's "Allow and continue" lands on `appStep1`.
- Step 4's "Submit application" routes into B's real `StepUpAuthScreen`; confirming there returns to `ApplicationConfirmationScreen`.
- B's session-expired demo (frame 12) dims the **real** `ApplicationStep2Screen` underneath the modal instead of a static mockup.

## New deviations for C — flagged
- **`SelectableTile`** — no selectable-card primitive in the DS; built from native radio/checkbox inputs styled as bordered tiles (same rationale as B's `RadioOption`).
- **`DeclarationRow`**'s Yes/No control — no two-option segmented primitive in the DS; built from tokens, consistent with the language switcher's existing segmented pattern.
- **`DocumentUploadCard`** — no file-upload primitive in the DS; built from tokens with a real (state-only, non-persisted) file input.
- **Licence-class dependency rule (C1 needs 2 years on B)** is static mock copy, not enforced logic, per product decision — the inline warning and "Remove and continue" action are wired for the one case shown in the wireframe (C1) but do not generalize to other class combinations.

## Wireframe B — the auth chain
`Landing catalogue → ServiceDetailScreen → IdentityLoginScreen → (FederatedIdpScreen →) ConsentScreen → appStep1`. Clicking any service card on the landing page opens its detail page; "Sign in to start application" enters the identity chain. Local sign-in and Mobile OTP grant **basic** assurance directly; National Digital ID / MOSIP route through the external eID stub and return **substantial** assurance — both raise the same `AuthContext` used by Wireframe A.

`identityService.ts` follows the same mock-latency + hook pattern as `portalService.ts`, in `useIdentityData.ts`.

**IS/eID chrome is intentionally not the portal chrome.** `IdentityShell` and `ExternalIdpShell` are separate layout wrappers (dashed "trust boundary" borders) — they don't reuse `AppHeader`/`AppFooter` because the identity server and the external eID provider are different products in the wireframe's own model.

**Frame 11 (step-up MFA) and frame 12 (session-expired modal)** are now wired for real: step 4 of the application routes through B's `StepUpAuthScreen`, and the session-expired demo overlays the real Step 2 screen. They remain additionally reachable via `DemoNav` for standalone review.

## New deviations for B — flagged
- **`RadioOption`** — the design system has no radio-button primitive (only `Checkbox` and `Switch`). Built as a native `<input type="radio">` styled with DS tokens.
- **`OtpInput`** — no code/OTP-input primitive in the DS. Built from DS tokens with per-digit inputs, paste support, and a single combined `aria-label` group for screen readers.
- **Consent Allow/Deny** — the wireframe draws this as a two-part segmented pill; implemented with the DS `Switch` instead (same boolean semantics, native DS component).
- **IdentityShell / ExternalIdpShell** — bespoke page chrome, built from DS tokens (no DS "auth shell" component exists).

## Wireframe A — design system fidelity
Every screen composes the DS's own `Button`, `Card`, `Badge`, `Alert`, `Input`, `Avatar`, `Breadcrumb`, `Logo` — copied as source (not rebuilt) into `src/design-system/`, importing only their design tokens (colors/spacing/typography/elevation) for anything not covered by a component. Icons use **Lucide** (per the design system's own guidance). The crest, citizen photograph, and credential QR code stay as clearly labeled dashed placeholder boxes, never icons, since they're real artwork the product doesn't have yet.

## New deviations for A — flagged
- **`LanguageSwitcher`** — no segmented-control primitive in the DS. Built from DS tokens. Copy is not translated yet.
- **`AccessibilityToolbar`** — visual-only stub per product decision.
- **No router** — a plain `Screen` state switch in `App.tsx` stands in for real routes. Every screen takes `onNavigate`, so swapping in `react-router` doesn't touch screen code.
- **STEP-UP / READY logic** — `stepUpRequired` per service is inferred from the wireframe's frame-2 annotations. Confirm against the real WSO2 Identity Server authorization policy once available.

## Backend-integration points
- `AuthContext` is a mock session (`signIn`/`signOut`/`raiseAssurance` just flip local state). Replace its internals with a real WSO2 Identity Server OIDC flow — the `{isAuthenticated, user, assuranceLevel}` shape consumed everywhere else doesn't need to change.
- Every `*Service.ts` file simulates network latency (and can simulate failure via `{ fail: true }`) around in-memory mock data shaped like a real API response. Swap each function's body for a real `fetch` call; hooks and screens are unaffected.
- Loading / error / empty states are all driven by the hook layer (`AsyncState` component) — not hardcoded per screen.
- `ConsentRow`'s revoke button, `revenueLicenceService.renewLicence`, and `adminConsoleService.submitForReview` are the concrete points where real write APIs plug in.

## Inconsistencies noticed across A–D (flag for cleanup before backend integration)
Called out now, at the end of the wireframe set, per the request to surface drift before real integration work starts:

1. **Hardcoded demo identity duplicated across services.** "John Doe" / "J. Doe", the NIC mask, and the DOB appear independently in `portalService.ts`, `identityService.ts`, `applicationService.ts`, and `sessionInspectorService.ts`. In a real integration these all come from one userinfo/ID-token response — worth extracting to a single mock-identity fixture the services share, so a later swap to real claims only touches one place.
2. **Two ways of building a segmented/pill control** exist (`LanguageSwitcher`'s inline implementation vs. the payment-method radio cards in `ApplicationStep4Screen` vs. `AssuranceLevelPicker`) — all solve the same "the DS has no segmented control" gap independently. Worth consolidating into one shared `SegmentedControl` primitive now that a third and fourth use case (assurance level, Yes/No declarations) have appeared.
3. **`Screen` union in `App.tsx` has grown to 20 values in one flat switch.** Fine for a demo, but naming isn't fully consistent (`appStep1..4` vs `applicationConfirmation`/`applicationError` vs `vehicleRevenueLicence`) — a real router migration is a good time to rename onto one scheme (e.g. all `application*`).
4. **Fee currency was briefly LKR then reverted to `$`** during earlier iteration; confirmed consistent as `$` everywhere now, but the mock data has no `currency` field — a real integration should carry currency explicitly rather than baking the symbol into strings.
5. **`MicroAppHeader` was duplicated per micro app until this pass** (Driving Licence Service hardcoded); generalized with `appName`/`tagline` props when Document D needed a second micro app. Worth checking `AppHeader` (portal) and `IdentityShell` (auth) for the same drift if a third context appears.
6. **`revenueLicenceService.renewLicence`'s return value (`receiptRef`) is currently unused** by `VehicleRevenueLicenceScreen` — the wireframe's "kept deliberately thin" note means no receipt UI exists yet; flag if a receipt/confirmation becomes in-scope later.
7. **Session inspector claims are hand-authored per client** in `sessionInspectorService.ts` rather than derived from a single source of truth for "what this session's ID token contains" — fine for two clients, would not scale past a handful without a shared claims model.

## Not built here (explicitly out of scope)
Mobile app wireframes — a separate, later phase. All four web wireframes (A–D) are implemented. The consent screen's "Deny" remains visual-only (no blocking flow) per earlier product decision, and the deferred-auth variant (frame 23) is intentionally not wired into the canonical flow, per the wireframe's own framing.
