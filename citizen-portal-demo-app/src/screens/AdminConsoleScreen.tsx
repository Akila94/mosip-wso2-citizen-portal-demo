import React, { useEffect, useState } from 'react';
import { AdminSidebarNav } from '../components/platform/AdminSidebarNav';
import { ScopeChecklist } from '../components/platform/ScopeChecklist';
import { AssuranceLevelPicker } from '../components/platform/AssuranceLevelPicker';
import { AsyncState } from '../components/common/AsyncState';
import { Input } from '../design-system/components/forms/Input';
import { Button } from '../design-system/components/core/Button';
import { Badge } from '../design-system/components/core/Badge';
import { Placeholder } from '../components/common/Placeholder';
import { useAdminNav, useScopeCatalogue, useServiceDraft } from '../hooks/useAdminConsoleData';
import { adminConsoleService } from '../services/adminConsoleService';
import type { AssuranceRequirement, ServiceDraft } from '../services/types';

/**
 * Platform console (frame 20) — a different persona (agency admin), not the
 * citizen portal, so it deliberately does not reuse AppHeader/MicroAppHeader.
 * Optional per the wireframe ("cut second if time is short"); reachable via
 * the demo nav rather than the citizen click-path.
 */
export function AdminConsoleScreen() {
  const { data: navItems } = useAdminNav();
  const { data: scopes, loading: scopesLoading, error: scopesError, reload: reloadScopes } = useScopeCatalogue();
  const { data: draftData, loading: draftLoading, error: draftError, reload: reloadDraft } = useServiceDraft();
  const [draft, setDraft] = useState<ServiceDraft | null>(null);
  const [saving, setSaving] = useState<'idle' | 'draft' | 'submit' | 'saved' | 'submitted'>('idle');

  useEffect(() => {
    if (draftData) setDraft(draftData);
  }, [draftData]);

  function toggleScope(id: string) {
    setDraft((d) => (d ? { ...d, selectedScopeIds: d.selectedScopeIds.includes(id) ? d.selectedScopeIds.filter((s) => s !== id) : [...d.selectedScopeIds, id] } : d));
  }

  function setAssurance(v: AssuranceRequirement) {
    setDraft((d) => (d ? { ...d, assuranceRequirement: v } : d));
  }

  async function handleSaveDraft() {
    if (!draft) return;
    setSaving('draft');
    await adminConsoleService.saveDraft(draft);
    setSaving('saved');
  }

  async function handleSubmit() {
    if (!draft) return;
    setSaving('submit');
    await adminConsoleService.submitForReview(draft);
    setSaving('submitted');
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 'var(--space-3) var(--space-8)', borderBottom: '1px solid var(--border-default)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)' }}>
          <span style={{ font: '600 13.5px var(--font-sans)' }}>MaroliaGov — platform console</span>
          <span style={{ font: '400 12px var(--font-mono)', color: 'var(--text-secondary)' }}>Applications / Register a service</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-3)' }}>
          <Badge tone="neutral">ROLE · AGENCY ADMIN</Badge>
          <Placeholder label="avatar" width={28} height={28} shape="circle" />
        </div>
      </header>
      <div style={{ display: 'flex', flex: 1, alignItems: 'stretch' }}>
        {navItems && <AdminSidebarNav items={navItems} activeId="applications" />}
        <main style={{ flex: 1, padding: 'var(--space-8)', display: 'flex', flexDirection: 'column', gap: 'var(--space-6)', maxWidth: 1100 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <h1 style={{ margin: 0, font: '700 24px var(--font-display)' }}>Register "Birth Certificate Service"</h1>
            <span style={{ font: '400 12.5px var(--font-sans)', color: 'var(--text-secondary)' }}>Everything an agency must supply to appear as a micro app in the citizen catalogue.</span>
          </div>

          <AsyncState loading={draftLoading} error={draftError} onRetry={reloadDraft}>
            {draft && (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                  <Input label="Service name (citizen-facing)" defaultValue={draft.serviceName} hint="Appears on the catalogue card and every consent screen." onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDraft({ ...draft, serviceName: e.target.value })} />
                  <Input label="Owning agency" defaultValue={draft.owningAgency} hint='Shown to citizens as "operated by".' onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDraft({ ...draft, owningAgency: e.target.value })} />
                  <Input label="Redirect URIs" defaultValue={draft.redirectUris} hint="One per line. Exact match enforced." onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDraft({ ...draft, redirectUris: e.target.value })} />
                  <Input label="Post-logout redirect" defaultValue={draft.postLogoutRedirect} hint="Used by single logout across all micro apps." onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDraft({ ...draft, postLogoutRedirect: e.target.value })} />
                </div>

                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 'var(--space-4)' }}>
                  <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)' }}>
                    <h2 style={{ margin: '0 0 var(--space-3)', font: '600 14px var(--font-sans)' }}>Requested scopes</h2>
                    <AsyncState loading={scopesLoading} error={scopesError} onRetry={reloadScopes}>
                      {scopes && <ScopeChecklist scopes={scopes} selectedIds={draft.selectedScopeIds} onToggle={toggleScope} />}
                    </AsyncState>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
                    <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                      <h2 style={{ margin: 0, font: '600 14px var(--font-sans)' }}>Assurance level required</h2>
                      <AssuranceLevelPicker value={draft.assuranceRequirement} onChange={setAssurance} />
                      <span style={{ font: '400 11.5px/1.5 var(--font-sans)', color: 'var(--text-secondary)' }}>
                        Substantial means a citizen on a password-only session is stepped up before submission, not blocked at the door.
                      </span>
                    </div>
                    <div style={{ border: '1.5px solid var(--border-strong)', borderRadius: 'var(--radius-md)', padding: 'var(--space-5)', display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
                      <h2 style={{ margin: 0, font: '600 14px var(--font-sans)' }}>Consent text citizens will see</h2>
                      <div style={{ border: '1px solid var(--border-default)', borderRadius: 'var(--radius-sm)', padding: 'var(--space-3)', background: 'var(--surface-sunken)', font: '400 11.5px/1.6 var(--font-sans)' }}>
                        "{draft.consentPreview}"
                      </div>
                      <span style={{ font: '400 11px var(--font-sans)', color: 'var(--text-secondary)' }}>Plain-language, reviewed by the portal team before the service goes live. 3 languages required.</span>
                    </div>
                  </div>
                </div>

                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: '1.5px solid var(--border-default)', paddingTop: 'var(--space-5)' }}>
                  <span style={{ font: '400 11.5px var(--font-mono)', color: 'var(--text-secondary)' }}>
                    status · {saving === 'submitted' || draft.status === 'submitted' ? 'submitted for review' : saving === 'saved' ? 'draft saved' : 'draft'} · not visible in the citizen catalogue
                  </span>
                  <div style={{ display: 'flex', gap: 'var(--space-2)' }}>
                    <Button variant="secondary" onClick={handleSaveDraft} disabled={saving === 'draft'}>{saving === 'draft' ? 'Saving…' : 'Save draft'}</Button>
                    <Button onClick={handleSubmit} disabled={saving === 'submit' || saving === 'submitted'}>{saving === 'submit' ? 'Submitting…' : saving === 'submitted' ? 'Submitted ✓' : 'Submit for review'}</Button>
                  </div>
                </div>
              </>
            )}
          </AsyncState>
        </main>
      </div>
    </div>
  );
}
