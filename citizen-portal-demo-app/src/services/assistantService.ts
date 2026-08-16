import type { AssistantMessage, ServiceRequestOptions } from './types';

const DEFAULT_DELAY_MS = 700;

function simulate<T>(payload: T, opts: ServiceRequestOptions = {}): Promise<T> {
  const { delayMs = DEFAULT_DELAY_MS, fail = false } = opts;
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error('The assistant is unavailable right now.'));
      else resolve(payload);
    }, delayMs);
  });
}

/**
 * Canned demo replies matching the wireframe's exact exchange. A real
 * integration would call an LLM grounded on the citizen's session + the
 * service catalogue (see the "Claude API in prototypes" pattern) — kept as
 * static mock responses here since frame 22 is explicitly optional/cut-first.
 */
export const assistantService = {
  ask(_question: string, isAuthenticated: boolean, opts?: ServiceRequestOptions): Promise<AssistantMessage> {
    if (!isAuthenticated) {
      return simulate(
        {
          id: `a-${Date.now()}`,
          role: 'assistant',
          text: 'I can answer general questions about services. Sign in for an answer about your specific licence.',
          actions: [{ label: 'Sign in', kind: 'link' }],
        },
        opts
      );
    }
    return simulate(
      {
        id: `a-${Date.now()}`,
        role: 'assistant',
        text: 'Your class B licence expires on 12 Sep 2026. To renew you need a medical fitness certificate issued within 3 months, and $18. You already have a verified identity, so nothing else.',
        actions: [
          { label: 'Start the renewal', kind: 'start-renewal' },
          { label: 'Where do I get a certificate?', kind: 'link' },
        ],
        sourceNote: 'answered from your record because you are signed in · sources: Transport registry, service catalogue',
      },
      opts
    );
  },
};
