import React from 'react';
import type { TestDay, TestSlotState } from '../../services/types';
import type { SelectedSlot } from '../../hooks/useTestSlotBooking';

const STATE_STYLE: Record<TestSlotState, { bg: string; fg: string; border: string }> = {
  held: { bg: 'var(--gray-950)', fg: 'var(--gray-0)', border: 'solid' },
  available: { bg: 'var(--surface-card)', fg: 'var(--text-primary)', border: 'solid' },
  full: { bg: 'var(--surface-sunken)', fg: 'var(--text-muted)', border: 'dashed' },
};

export interface TestSlotGridProps {
  week: TestDay[];
  weekLabel: string;
  selected: SelectedSlot | null;
  countdownLabel: string;
  secondsRemaining: number;
  office: string;
  onSelect: (day: string, time: string, state: TestSlotState) => void;
  onPrevWeek: () => void;
  onNextWeek: () => void;
  canGoPrev: boolean;
}

/** Real interactivity: clicking an available slot holds it (releasing any
 * previous hold) and starts a live 10-minute countdown; a full slot is
 * inert. Week navigation re-fetches from applicationService. */
export function TestSlotGrid({ week, weekLabel, selected, countdownLabel, secondsRemaining, office, onSelect, onPrevWeek, onNextWeek, canGoPrev }: TestSlotGridProps) {
  return (
    <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
      {selected ? (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 'var(--space-3)' }}>
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ font: '600 13.5px var(--font-sans)' }}>Slot held: {selected.day}, {selected.time}</span>
            <span style={{ font: '400 11.5px var(--font-sans)', color: 'var(--text-secondary)' }}>{office} · held while you finish this step. Release it and it goes back to the pool.</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
            <div role="timer" aria-live="off" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', border: `1.5px solid ${secondsRemaining <= 60 ? 'var(--danger)' : 'var(--border-strong)'}`, borderRadius: 'var(--radius-sm)', padding: '8px 14px' }}>
              <span style={{ font: '600 20px var(--font-mono)', color: secondsRemaining <= 60 ? 'var(--danger)' : 'var(--text-primary)' }}>{countdownLabel}</span>
              <span style={{ font: '500 9.5px var(--font-mono)', letterSpacing: '0.1em', color: 'var(--text-secondary)' }}>HOLD REMAINING</span>
            </div>
            <span aria-live="polite" className="sr-only" style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden', clip: 'rect(0 0 0 0)' }}>
              {secondsRemaining === 300 ? '5 minutes remaining on your held slot.' : secondsRemaining === 60 ? '1 minute remaining on your held slot.' : ''}
            </span>
          </div>
        </div>
      ) : (
        <div role="status" aria-live="polite" style={{ border: '1px dashed var(--border-strong)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>
          No slot held. Select an available time below to hold it for 10 minutes.
        </div>
      )}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ font: '600 13px var(--font-sans)' }}>Week of {weekLabel}</span>
        <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
          <button type="button" disabled={!canGoPrev} onClick={onPrevWeek} style={{ padding: '6px 12px', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', font: '500 11.5px var(--font-mono)', background: 'var(--surface-card)', cursor: canGoPrev ? 'pointer' : 'not-allowed', opacity: canGoPrev ? 1 : 0.5 }}>
            ‹ prev
          </button>
          <button type="button" onClick={onNextWeek} style={{ padding: '6px 12px', border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-sm)', font: '500 11.5px var(--font-mono)', background: 'var(--surface-card)', cursor: 'pointer' }}>
            next ›
          </button>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: `repeat(${week.length}, 1fr)`, gap: 'var(--space-3)' }}>
        {week.map((d) => (
          <div key={d.day} style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
            <div style={{ font: '600 12px var(--font-sans)', textAlign: 'center', borderBottom: '1px dashed var(--border-default)', paddingBottom: 6 }}>{d.day}</div>
            {d.slots.map((s, i) => {
              const style = STATE_STYLE[s.state];
              const full = s.state === 'full';
              return (
                <button
                  key={i}
                  type="button"
                  disabled={full}
                  aria-pressed={s.state === 'held'}
                  onClick={() => onSelect(d.day, s.time, s.state)}
                  style={{ border: `1.5px ${style.border} var(--border-strong)`, borderRadius: 'var(--radius-sm)', padding: 9, textAlign: 'center', font: '500 11.5px var(--font-mono)', background: style.bg, color: style.fg, cursor: full ? 'not-allowed' : 'pointer' }}
                >
                  {s.time}
                </button>
              );
            })}
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 'var(--space-4)', borderTop: '1px dashed var(--border-default)', paddingTop: 'var(--space-2)', font: '400 11px var(--font-mono)', color: 'var(--text-secondary)', flexWrap: 'wrap' }}>
        <span>■ selected &amp; held</span>
        <span>□ available</span>
        <span>▨ full</span>
        <span>only a few slots each week — scarcity is real, not decorative</span>
      </div>
    </div>
  );
}
