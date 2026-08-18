export type AssuranceLevel = 'none' | 'basic' | 'substantial';
export type ServiceState = 'LIVE' | 'STUB' | 'READY' | 'STEP_UP';
export type SignatureRequirement = 'none' | 'optional' | 'required';

export interface EffortContract {
  signInRequired: boolean;
  signature: SignatureRequirement;
  steps: number;
  timeEstimate: string;
  fee: string;
}

export interface ServiceItem {
  id: string;
  title: string;
  description: string;
  state: ServiceState;
  /** Whether submitting this service needs a higher assurance level than a plain sign-in (drives STEP_UP once authenticated). */
  stepUpRequired: boolean;
  effort: EffortContract;
}

export interface ServiceCategory {
  id: string;
  name: string;
  count: string;
  services: ServiceItem[];
}

export interface LifeEvent {
  id: string;
  title: string;
  serviceCount: string;
}

export type TimelineChip = 'Action needed' | 'Payment due' | 'Appointment' | 'Waiting on government' | 'For information';

export interface TimelineItem {
  id: string;
  date: string;
  relative: string;
  title: string;
  description: string;
  chip: TimelineChip;
  actionLabel: string;
}

export interface SubmittedApplication {
  id: string;
  reference: string;
  title: string;
  status: string;
}

export type AttributeSource = 'verified' | 'self-asserted';

export interface AttributeRecord {
  id: string;
  label: string;
  value: string;
  source: AttributeSource;
  sourceLabel: string;
  editable: boolean;
}

export interface DepartmentRecordRow { label: string; value: string; }

export interface DepartmentRecord {
  id: string;
  department: string;
  rows: DepartmentRecordRow[];
}

export interface ConsentGrant {
  id: string;
  appName: string;
  agency: string;
  scopes: string[];
  grantedDate: string;
}

export type WalletDocumentStatus = 'VALID' | 'NOT_ISSUED' | 'NEW';

export interface WalletDocument {
  id: string;
  title: string;
  status: WalletDocumentStatus;
  number: string;
  issuedDate: string;
  expiryDate: string;
  primaryAction: string;
  secondaryAction: string;
}

export interface ServiceDetailSection {
  id: string;
  heading: string;
  note: string;
  items: string[];
}

export interface ServiceChannel {
  id: string;
  title: string;
  description: string;
}

export interface ServiceCost {
  fee: string;
  processingTime: string;
  formLength: string;
  payment: string;
}

export interface ServiceDetail {
  id: string;
  title: string;
  category: string;
  department: string;
  summary: string;
  sections: ServiceDetailSection[];
  channels: ServiceChannel[];
  cost: ServiceCost;
  signInWith: string[];
}

export type IdpAssurance = 'basic' | 'substantial';

export interface IdentityProvider {
  id: string;
  name: string;
  description: string;
  assuranceLevel: IdpAssurance;
  /** True when this method hands off to an external provider stub (frame 9) before returning. */
  externalHop: boolean;
}

export interface ConsentAttributeRequest {
  id: string;
  label: string;
  required: boolean;
  value: string;
  source: string;
}

export interface AuthResult {
  assuranceLevel: AssuranceLevel;
}

export interface LicenceApplicationType {
  id: string;
  label: string;
  description: string;
}

export interface LicenceClass {
  id: string;
  label: string;
  description: string;
  eligible: boolean;
  ineligibleReason?: string;
}

export interface EditableFieldDef {
  id: string;
  label: string;
  required: boolean;
  value: string;
  help: string;
}

export interface VerifiedAttributeLine {
  label: string;
  value: string;
}

export interface VerifiedIdentitySummary {
  name: string;
  badgeLabel: string;
  attributes: VerifiedAttributeLine[];
}

export interface SelfAssertedAttribute {
  id: string;
  label: string;
  value: string;
}

export interface DeclarationQuestion {
  id: string;
  question: string;
  help: string;
  /** A "yes" here routes to the application error screen instead of blocking silently. */
  yesTriggersReview: boolean;
}

export type DeclarationAnswer = 'yes' | 'no' | null;

export type TestSlotState = 'held' | 'available' | 'full';

export interface TestSlot {
  time: string;
  state: TestSlotState;
}

export interface TestDay {
  day: string;
  slots: TestSlot[];
}

export interface FeeLine {
  label: string;
  amount: string;
}

export interface ReviewSection {
  step: string;
  lines: string[];
}

export interface ApplicationConfig {
  appTypes: LicenceApplicationType[];
  licenceClasses: LicenceClass[];
  permitNumber: string;
  permitIssueDate: string;
  editableFields: EditableFieldDef[];
  verifiedIdentity: VerifiedIdentitySummary;
  selfAsserted: SelfAssertedAttribute[];
  declarations: DeclarationQuestion[];
  collectionDistricts: string[];
  pickupStations: string[];
  feeBreakdown: FeeLine[];
  totalFee: string;
  review: ReviewSection[];
}

export interface ApplicationConfirmation {
  reference: string;
  paymentReference: string;
  amountDue: string;
  appointment: string;
  location: string;
  processingEstimate: string;
  nextSteps: string[];
}

export interface ApplicationErrorRoute {
  label: string;
  description: string;
}

export interface ApplicationErrorInfo {
  title: string;
  description: string;
  reasonHeading: string;
  reasonBody: string;
  reference: string;
  routes: ApplicationErrorRoute[];
}

export interface RegistryCheck {
  label: string;
  status: string;
}

export interface VehicleRecord {
  id: string;
  plate: string;
  description: string;
  dueDate: string;
  fee: string;
  owner: string;
  province: string;
  registryChecks: RegistryCheck[];
}

export interface SessionRow {
  label: string;
  value: string;
}

/** One app currently holding a session against the same IdP session (`sid`). */
export interface SessionInspectorClient {
  appKey: string;
  appName: string;
}

/**
 * The session inspector's data, as returned by `/bff/{app}/api/session-inspector`.
 *
 * These are facts about the real session, not display rows: the panel composes
 * its own labels and its own live countdown from `expiresAt`. `releasedClaims`
 * is the actual ID-token claim set this client received, which is what makes
 * the side-by-side comparison between the two micro apps meaningful — it is
 * `Record<string, unknown>` rather than `Record<string, string | string[]>`
 * because real claims include numbers (`exp`, `iat`, `auth_time`).
 */
export interface SessionInspectorData {
  appKey: string;
  clientId: string;
  clientLabel: string;
  subject: string;
  idp: string;
  acr?: string;
  amr: string[];
  sid: string;
  assuranceLevel: AssuranceLevel;
  /** Seconds since the Unix epoch. */
  authTime: number;
  /** Seconds since the Unix epoch. */
  expiresAt: number;
  clientsInSession: SessionInspectorClient[];
  releasedClaims: Record<string, unknown>;
}

/**
 * The citizen's record as held by the government registry, keyed server-side
 * by the verified OIDC subject and **projected by the calling app's scopes** —
 * so the same citizen legitimately yields different fields to different micro
 * apps. Absent fields mean "this app was not released that claim", never
 * "unknown".
 */
export interface CitizenProfile {
  sub: string;
  name?: string;
  birthdate?: string;
  nic?: string;
  address?: string;
}

export type AssuranceRequirement = 'low' | 'substantial' | 'high';

export interface AdminNavItem {
  id: string;
  label: string;
}

export interface RegistrationFieldDef {
  id: string;
  label: string;
  value: string;
  help: string;
}

export interface ScopeOption {
  id: string;
  label: string;
  description: string;
  requiredByDefault: boolean;
}

export interface ServiceDraft {
  serviceName: string;
  owningAgency: string;
  redirectUris: string;
  postLogoutRedirect: string;
  selectedScopeIds: string[];
  assuranceRequirement: AssuranceRequirement;
  consentPreview: string;
  status: 'draft' | 'submitted';
}

export interface AssistantMessage {
  id: string;
  role: 'user' | 'assistant';
  text: string;
  actions?: { label: string; kind: 'start-renewal' | 'link' }[];
  sourceNote?: string;
}

export interface ServiceRequestOptions {
  /** Simulated network latency in ms. */
  delayMs?: number;
  /** Force this call to reject, to exercise error states. */
  fail?: boolean;
}
