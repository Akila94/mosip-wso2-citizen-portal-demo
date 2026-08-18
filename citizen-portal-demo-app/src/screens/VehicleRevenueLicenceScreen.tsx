import React, { useState } from 'react';
import { MicroAppHeader } from '../components/application/MicroAppHeader';
import { VerifiedIdentityPanel } from '../components/application/VerifiedIdentityPanel';
import { VehicleCard } from '../components/revenue/VehicleCard';
import { SessionInspectorToggle } from '../components/session/SessionInspectorToggle';
import { SessionInspectorPanel } from '../components/session/SessionInspectorPanel';
import { AsyncState } from '../components/common/AsyncState';
import { AppFooter } from '../components/layout/AppFooter';
import { Alert } from '../design-system/components/feedback/Alert';
import { useVehicles, useRevenueLicenceIdentity } from '../hooks/useRevenueLicenceData';
import { revenueLicenceService } from '../services/revenueLicenceService';
import styles from '../styles/layout.module.css';

/**
 * Micro app B — Document D's SSO payoff (frame 19), never cut. Deliberately
 * thin: one callout's worth of content (a vehicle card and two registry
 * checks) — a form here would bury the point, which is that nothing on
 * this screen required a new sign-in or a re-typed plate number.
 */
export function VehicleRevenueLicenceScreen({ onSaveExit }: { onSaveExit: () => void }) {
  const { data: vehicles, loading, error, reload } = useVehicles();
  const identity = useRevenueLicenceIdentity();
  const [renewingId, setRenewingId] = useState<string | null>(null);
  const [renewedIds, setRenewedIds] = useState<string[]>([]);
  const [renewError, setRenewError] = useState<string | null>(null);
  const [inspectorOpen, setInspectorOpen] = useState(false);

  async function handleRenew(vehicleId: string) {
    setRenewingId(vehicleId);
    setRenewError(null);
    try {
      await revenueLicenceService.renewLicence(vehicleId);
      setRenewedIds((ids) => [...ids, vehicleId]);
      // The renewal is recorded server-side, so re-read rather than patching
      // local state — the card's new due date comes from the registry.
      reload();
    } catch (err) {
      setRenewError(err instanceof Error ? err.message : 'Could not renew this licence.');
    } finally {
      setRenewingId(null);
    }
  }

  return (
    <>
      <MicroAppHeader onSaveExit={onSaveExit} appName="Vehicle Revenue Licence" tagline="a different micro app · Provincial Revenue Office" />
      <AsyncState loading={loading} error={error} onRetry={reload} loadingLabel="Loading your vehicles…">
        {vehicles && (
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <div className={styles.page} style={{ flex: 1, margin: 0, maxWidth: 'none' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Renew a revenue licence</h1>
                <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>Two steps · ~4 min · from $15.</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 'var(--space-2)' }}>
                  <h2 style={{ margin: 0, font: '600 14px var(--font-sans)' }}>Your vehicles</h2>
                  <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>already pulled from your profile — nothing typed</span>
                </div>
                {renewError && <Alert tone="danger">{renewError}</Alert>}
                <AsyncState loading={false} error={null} isEmpty={vehicles.length === 0} emptyMessage="No vehicles registered to your verified NIC.">
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                    {vehicles.map((v) => (
                      <VehicleCard key={v.id} vehicle={v} onRenew={() => handleRenew(v.id)} renewing={renewingId === v.id} renewed={renewedIds.includes(v.id)} />
                    ))}
                    {vehicles.length < 2 && (
                      <div style={{ border: '1.5px dashed var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 'var(--space-2)', background: 'var(--surface-sunken)' }}>
                        <span style={{ font: '600 13px var(--font-sans)', color: 'var(--text-secondary)' }}>No other vehicles registered</span>
                        <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-muted)' }}>
                          Registered vehicles come from the Transport registry against your verified NIC — the citizen never types a plate number.
                        </span>
                      </div>
                    )}
                  </div>
                </AsyncState>
              </div>
            </div>
            <div style={{ width: 360, flexShrink: 0, padding: 'var(--space-6)', background: 'var(--surface-sunken)', borderLeft: '1px solid var(--border-default)', minHeight: '100%', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
              <AsyncState loading={identity.loading} error={identity.error} onRetry={identity.reload} loadingLabel="Loading your verified identity…">
                {identity.data && <VerifiedIdentityPanel identity={identity.data} />}
              </AsyncState>
              <div style={{ border: '1px dotted var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-4)', font: '400 11.5px/1.55 var(--font-sans)', color: 'var(--text-primary)' }}>
                <strong>Say this out loud in the demo:</strong> two services, two agencies, two consent grants — one sign-in. Then open the session inspector to show the same session id behind both.
              </div>
            </div>
          </div>
        )}
      </AsyncState>
      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', padding: 'var(--space-4) var(--space-8)', borderTop: '1px solid var(--border-default)', background: 'var(--surface-sunken)' }}>
        <span style={{ font: '400 11px var(--font-mono)', color: 'var(--text-secondary)' }}>same session as Driving Licence Service</span>
        <SessionInspectorToggle open={inspectorOpen} onToggle={() => setInspectorOpen((v) => !v)} />
      </div>
      {inspectorOpen && <SessionInspectorPanel appKey="revenue-licence" onCollapse={() => setInspectorOpen(false)} />}
      <AppFooter compact />
    </>
  );
}
