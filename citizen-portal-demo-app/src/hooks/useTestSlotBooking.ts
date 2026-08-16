import { useEffect, useRef, useState } from 'react';
import { applicationService } from '../services/applicationService';
import type { TestDay, TestSlotState } from '../services/types';

const HOLD_SECONDS = 10 * 60; // real 10-minute hold, per product decision

export interface SelectedSlot {
  day: string;
  time: string;
}

/**
 * Live test-slot booking: fetches a week of slots from applicationService
 * (simulated latency, like the rest of the mock service layer), lets the
 * citizen page between weeks, hold an available slot, and counts the hold
 * down in real time with setInterval — releasing it automatically at 0.
 */
export function useTestSlotBooking() {
  const [weekOffset, setWeekOffset] = useState(0);
  const [week, setWeek] = useState<TestDay[]>([]);
  const [weekLabel, setWeekLabel] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<SelectedSlot | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState(0);
  const requestId = useRef(0);

  useEffect(() => {
    const id = ++requestId.current;
    setLoading(true);
    setError(null);
    const base = new Date(2026, 7, 18);
    const monday = new Date(base);
    monday.setDate(base.getDate() + weekOffset * 7);
    const friday = new Date(monday);
    friday.setDate(monday.getDate() + 4);
    const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    applicationService
      .getTestWeek(weekOffset)
      .then((data) => {
        if (id === requestId.current) {
          setWeek(data);
          setWeekLabel(`${monday.getDate()} ${monthNames[monday.getMonth()]} \u2013 ${friday.getDate()} ${monthNames[friday.getMonth()]} 2026`);
          setLoading(false);
        }
      })
      .catch((err: Error) => {
        if (id === requestId.current) {
          setError(err.message || 'Could not load test slots.');
          setLoading(false);
        }
      });
  }, [weekOffset]);

  useEffect(() => {
    if (!selected || secondsRemaining <= 0) return;
    const interval = window.setInterval(() => {
      setSecondsRemaining((s) => {
        if (s <= 1) {
          setSelected(null);
          return 0;
        }
        return s - 1;
      });
    }, 1000);
    return () => window.clearInterval(interval);
  }, [selected, secondsRemaining > 0]);

  function selectSlot(day: string, time: string, state: TestSlotState) {
    if (state === 'full') return;
    setSelected({ day, time });
    setSecondsRemaining(HOLD_SECONDS);
  }

  function releaseSlot() {
    setSelected(null);
    setSecondsRemaining(0);
  }

  function changeWeek(delta: number) {
    setWeekOffset((w) => Math.max(0, w + delta));
  }

  const displayWeek: TestDay[] = week.map((d) => ({
    ...d,
    slots: d.slots.map((s) => (selected && selected.day === d.day && selected.time === s.time ? { ...s, state: 'held' as TestSlotState } : s)),
  }));

  const minutes = Math.floor(secondsRemaining / 60);
  const seconds = secondsRemaining % 60;
  const countdownLabel = `${minutes}:${seconds.toString().padStart(2, '0')}`;

  return {
    week: displayWeek,
    weekLabel,
    loading,
    error,
    reload: () => setWeekOffset((w) => w),
    weekOffset,
    changeWeek,
    selected,
    secondsRemaining,
    countdownLabel,
    selectSlot,
    releaseSlot,
    expired: secondsRemaining === 0 && !selected,
  };
}
