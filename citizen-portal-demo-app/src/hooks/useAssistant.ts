import { useState } from 'react';
import { assistantService } from '../services/assistantService';
import type { AssistantMessage } from '../services/types';

/**
 * Docked assistant's chat state. Interactive/user-driven (messages sent,
 * open/closed), so it lives in a hook rather than being fetched — each
 * reply comes from assistantService, following the same call-through-a-hook
 * rule as every other service in this app.
 */
export function useAssistant(isAuthenticated: boolean) {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<AssistantMessage[]>([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function send(question: string) {
    if (!question.trim()) return;
    const userMsg: AssistantMessage = { id: `u-${Date.now()}`, role: 'user', text: question };
    setMessages((m) => [...m, userMsg]);
    setSending(true);
    setError(null);
    try {
      const reply = await assistantService.ask(question, isAuthenticated);
      setMessages((m) => [...m, reply]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong.');
    } finally {
      setSending(false);
    }
  }

  return { open, setOpen, messages, sending, error, send };
}
