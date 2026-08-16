import type {
  ApplicationConfig,
  ApplicationConfirmation,
  ApplicationErrorInfo,
  ServiceRequestOptions,
} from './types';

const DEFAULT_DELAY_MS = 500;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('Could not reach the application service. Please try again.'));
      else resolve(payload);
    }, delayMs);
  });
}

const config: ApplicationConfig = {
  appTypes: [
    { id: 'new', label: 'New licence', description: 'first licence for this class' },
    { id: 'renewal', label: 'Renewal', description: 'within 6 months of expiry' },
    { id: 'duplicate', label: 'Duplicate', description: 'lost or damaged' },
  ],
  licenceClasses: [
    { id: 'a1', label: 'A1 — motorcycle ≤125cc', description: 'from age 16', eligible: true },
    { id: 'a', label: 'A — motorcycle', description: 'from age 18', eligible: true },
    { id: 'b1', label: 'B1 — light tricycle', description: 'from age 18', eligible: true },
    { id: 'b', label: 'B — motor car', description: 'from age 18', eligible: true },
    { id: 'c1', label: 'C1 — light lorry', description: 'class B held 2 years', eligible: false, ineligibleReason: 'C1 requires a class B licence held for 2 years. You can apply for C1 from 13 Aug 2028, or continue with class B now.' },
    { id: 'd1', label: 'D1 — light bus', description: 'class B held 3 years', eligible: true },
  ],
  permitNumber: 'LP-2026-88214',
  permitIssueDate: '02 Apr 2026',
  editableFields: [
    { id: 'contact', label: 'Contact number', required: true, value: '+94 7•• ••• 220', help: 'Used for test reminders and OTP. Verified by code when changed.' },
    { id: 'email', label: 'Email address', required: true, value: 'john.doe@example.mr', help: 'Where the acknowledgement PDF is sent.' },
    { id: 'bloodGroup', label: 'Blood group', required: true, value: 'O+', help: 'Printed on the licence. Ask your clinic if unsure.' },
    { id: 'emergencyContact', label: 'Emergency contact', required: true, value: 'M. Doe · +94 7•• ••• 118', help: 'Name and number. Not shared with any other service.' },
  ],
  verifiedIdentity: {
    name: 'John Doe',
    badgeLabel: 'VERIFIED — NATIONAL DIGITAL ID',
    attributes: [
      { label: 'NIC number', value: '19•• ••• •••• 4471' },
      { label: 'Date of birth', value: '04 Mar 1996' },
      { label: 'Address', value: '14 Lake Road, Marolia City' },
      { label: 'Released to', value: 'Driving Licence Service' },
    ],
  },
  selfAsserted: [
    { id: 'contact', label: 'Mobile number', value: '+94 7•• ••• 220' },
    { id: 'email', label: 'Email', value: 'john.doe@example.mr' },
    { id: 'bloodGroup', label: 'Blood group', value: 'O+' },
  ],
  declarations: [
    { id: 'vision', question: 'Do you need corrected vision to drive?', help: 'Glasses or contact lenses. Adds a condition code to your licence.', yesTriggersReview: false },
    { id: 'blackouts', question: 'Have you had epilepsy, seizures or blackouts?', help: 'In the last 5 years, treated or untreated.', yesTriggersReview: true },
    { id: 'colourBlind', question: 'Do you have any colour blindness?', help: 'Red-green deficiency is assessed, not automatically disqualifying.', yesTriggersReview: false },
  ],
  collectionDistricts: ['Marolia Central', 'Marolia West', 'Marolia North', 'Marolia South'],
  pickupStations: ['Marolia West licence office', 'Marolia Central licence office'],
  feeBreakdown: [
    { label: 'Licence issue fee — class B', amount: '$14' },
    { label: 'Written test fee', amount: '$4' },
    { label: 'Card production', amount: '$2' },
    { label: 'Postal delivery', amount: 'not selected' },
  ],
  totalFee: '$20',
  review: [
    { step: 'Step 1', lines: ['New licence · class B (motor car)', 'Learner permit LP-2026-88214, issued 02 Apr 2026', 'Eligibility: passed'] },
    { step: 'Step 2', lines: ['+94 7•• ••• 220 · john.doe@example.mr · O+', 'Emergency contact: M. Doe', 'Collect at Marolia West licence office, Marolia Central'] },
    { step: 'Step 3', lines: ['Corrected vision: yes → condition code 01', 'Epilepsy or blackouts: no · Colour blindness: no', 'Medical certificate and learner permit uploaded'] },
    { step: 'Appointment', lines: ['Written test · Thu 20 Aug 2026, 09:30', 'Marolia West licence office'] },
  ],
};

const medicalReviewError: ApplicationErrorInfo = {
  title: "We can't continue online — a medical review is needed first",
  description: 'Your application is saved. It is not refused.',
  reasonHeading: 'The specific reason',
  reasonBody: 'You declared a history of blackouts. Regulation 14(3) requires a specialist assessment before a class B licence can be issued, whatever the rest of the application says.',
  reference: 'ref DL-2026-004871 · saved 14:38 · valid 90 days',
  routes: [
    { label: 'Book a specialist assessment', description: 'Any state clinic or approved specialist. The report is uploaded against this reference — you do not restart the application.' },
    { label: 'Continue without class B', description: 'Class A1 motorcycle is unaffected by regulation 14(3). You can amend step 1 and proceed today.' },
    { label: 'Talk to someone', description: 'Assisted agents at 42 service centres, or call 1500. Quote reference DL-2026-004871.' },
  ],
};

export const applicationService = {
  getApplicationConfig(opts?: ServiceRequestOptions) {
    return simulate(config, opts);
  },

  /** Deterministic mock week generator — a real backend would query the
   * test-centre scheduling system for `weekOffset` weeks from today.
   * Monday of week 0 is fixed to 18 Aug 2026 to match the wireframe. */
  getTestWeek(weekOffset: number, opts?: ServiceRequestOptions): Promise<import('./types').TestDay[]> {
    const base = new Date(2026, 7, 18); // 18 Aug 2026, a Monday
    const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri'];
    const times = ['09:30', '11:00', '14:00'];
    const week = days.map((label, dayIdx) => {
      const date = new Date(base);
      date.setDate(base.getDate() + weekOffset * 7 + dayIdx);
      const slots = times.map((time, timeIdx) => {
        const seed = (weekOffset * 17 + dayIdx * 5 + timeIdx * 3) % 5;
        return { time, state: (seed === 0 ? 'full' : 'available') as import('./types').TestSlotState };
      });
      return { day: `${label} ${date.getDate()}`, slots };
    });
    return simulate(week, opts);
  },

  submitApplication(_payload: unknown, opts?: ServiceRequestOptions): Promise<ApplicationConfirmation> {
    return simulate(
      {
        reference: 'DL-2026-004871',
        paymentReference: 'PAY-DL-771204 · valid until 20 Aug 2026',
        amountDue: '$20 — unpaid',
        appointment: 'Written test · Thu 20 Aug 2026, 09:30',
        location: 'Marolia West licence office',
        processingEstimate: '10 working days after a passed test',
        nextSteps: [
          '1 — Pay $20 now or with the reference above at any bank, post office, agent or by USSD *363#.',
          '2 — Bring your national identity card to the test. Nothing printed is required.',
          '3 — Results are recorded against this reference; watch My Timeline.',
          '4 — Your licence appears in My Documents and at your chosen pick-up station.',
        ],
      },
      opts
    );
  },

  getMedicalReviewError(opts?: ServiceRequestOptions) {
    return simulate(medicalReviewError, opts);
  },
};
