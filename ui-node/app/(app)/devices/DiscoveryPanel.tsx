
'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';

import type { DiscoveryStatus } from './types';
import { initialDiscoveryRunState } from './state';
import { triggerDiscovery } from './actions';
import { api } from '../../../lib/api-client';
import { Card, CardBody } from '../../_components/ui/Card';
import { Field, Hint, Label } from '../../_components/ui/Field';
import { Input, Select } from '../../_components/ui/Inputs';
import { Button } from '../../_components/ui/Button';
import { Badge } from '../../_components/ui/Badge';
import { Alert } from '../../_components/ui/Alert';
import { getDiscoveryStatusBadgeTone } from '../discovery/status';
import { ConfirmDialog } from '../../_components/ui/ConfirmDialog';
import { getScanPresetLabel, SCAN_PRESET_OPTIONS } from '../discovery/presets';
import { formatScanTags, normalizeScanTags, SCAN_TAG_OPTIONS, type ScanTag } from '../discovery/tags';

type Props = {
  status: DiscoveryStatus;
  readOnly?: boolean;
};

function formatTimestamp(ts?: string | null) {
  if (!ts) return 'n/a';
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return ts;
  }
  return d.toLocaleString();
}

function formatDurationMs(ms: number) {
  if (!Number.isFinite(ms) || ms < 0) return '—';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes <= 0) return `${seconds}s`;
  return `${minutes}m ${seconds}s`;
}

function formatLastUpdated(updatedAtMs: number) {
  if (!Number.isFinite(updatedAtMs) || updatedAtMs <= 0) return '—';
  const deltaMs = Date.now() - updatedAtMs;
  if (deltaMs < 2000) return 'just now';
  if (deltaMs < 60_000) return `${Math.round(deltaMs / 1000)}s ago`;
  if (deltaMs < 60 * 60_000) return `${Math.round(deltaMs / 60_000)}m ago`;
  return new Date(updatedAtMs).toLocaleString();
}

export function DiscoveryPanel({ status, readOnly = false }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(triggerDiscovery, initialDiscoveryRunState());
  const [liveStatus, setLiveStatus] = useState<DiscoveryStatus>(status);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const bypassConfirmRef = useRef(false);
  const formRef = useRef<HTMLFormElement | null>(null);
  const initialRun = status.latest_run ?? undefined;
  const initialTags = useMemo(() => {
    const rawStats = initialRun?.stats;
    const rawTags =
      rawStats && typeof rawStats === 'object' ? (rawStats as Record<string, unknown>).tags : undefined;
    if (!Array.isArray(rawTags)) {
      return [] as ScanTag[];
    }
    return normalizeScanTags(rawTags as FormDataEntryValue[]);
  }, [initialRun?.stats]);
  const [scopeValue, setScopeValue] = useState(() => (typeof initialRun?.scope === 'string' ? initialRun.scope : ''));
  const [selectedTags, setSelectedTags] = useState<ScanTag[]>(() => initialTags);

  const statusQuery = useQuery({
    queryKey: ['discovery-status'],
    initialData: status,
    queryFn: async ({ signal }) => {
      const res = await api.GET('/v1/discovery/status', {
        signal,
        headers: {
          'X-Request-ID': globalThis.crypto?.randomUUID?.()
        }
      });

      if (res.error) {
        throw new Error('Failed to fetch discovery status.');
      }
      return (res.data ?? status) as DiscoveryStatus;
    },
    refetchInterval: (query) => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        return false;
      }
      // Slow down polling when errors occur so we don't hammer the API.
      return query.state.status === 'error' ? 30_000 : 10_000;
    },
    refetchIntervalInBackground: false
  });

  const scopeSuggestionsQuery = useQuery({
    queryKey: ['discovery-scope-suggestions'],
    queryFn: async ({ signal }) => {
      const res = await api.GET('/v1/discovery/scope-suggestions', {
        signal,
        headers: {
          'X-Request-ID': globalThis.crypto?.randomUUID?.()
        }
      });
      if (res.error) {
        throw new Error('Failed to fetch scope suggestions.');
      }
      return (res.data ?? { scopes: [] }) as { scopes: Array<{ scope: string; interface?: string | null; address?: string | null }> };
    },
    staleTime: 5 * 60_000,
    refetchOnWindowFocus: false
  });

  const latest = statusQuery.data.latest_run ?? undefined;
  const latestPreset = latest?.stats && typeof latest.stats === 'object'
    ? (latest.stats as Record<string, unknown>).preset
    : undefined;
  const latestTags = latest?.stats && typeof latest.stats === 'object'
    ? (latest.stats as Record<string, unknown>).tags
    : undefined;
  const latestTagLabel = formatScanTags(latestTags);
  const defaultPreset = typeof latestPreset === 'string' && ['fast', 'normal', 'deep'].includes(latestPreset.trim().toLowerCase())
    ? latestPreset.trim().toLowerCase()
    : 'normal';

  const inProgress = liveStatus.status === 'queued' || liveStatus.status === 'running';
  const canTrigger = !readOnly;
  const confirmNeeded = canTrigger && inProgress;

  const scopeSuggestions = scopeSuggestionsQuery.data?.scopes ?? [];

  const progressLabel = useMemo(() => {
    if (!latest) return null;
    if (!latest.started_at) return null;
    const started = Date.parse(latest.started_at);
    if (!Number.isFinite(started)) return null;
    const end = latest.completed_at ? Date.parse(latest.completed_at) : Date.now();
    if (!Number.isFinite(end)) return null;
    const elapsed = end - started;
    const prefix = latest.completed_at ? 'Duration' : 'Elapsed';
    return `${prefix}: ${formatDurationMs(elapsed)}`;
  }, [latest]);

  const latestSummary = useMemo(() => {
    const stats = latest?.stats;
    if (!stats || typeof stats !== 'object') {
      return null;
    }
    const record = stats as Record<string, unknown>;
    const stage = typeof record.stage === 'string' ? record.stage : '';
    if (stage !== 'completed') {
      return null;
    }
    const seen = typeof record.devices_seen === 'number' ? record.devices_seen : null;
    const created = typeof record.devices_created === 'number' ? record.devices_created : null;
    const arpEntries = typeof record.arp_entries === 'number' ? record.arp_entries : null;
    const parts = [
      seen !== null ? `Devices seen: ${seen}` : null,
      created !== null ? `Created: ${created}` : null,
      arpEntries !== null ? `ARP entries: ${arpEntries}` : null
    ].filter(Boolean);
    return parts.length > 0 ? parts.join(' • ') : null;
  }, [latest?.stats]);

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
    }
  }, [state.status, router]);

  useEffect(() => {
    setLiveStatus(statusQuery.data);
  }, [statusQuery.data]);

  const badgeTone = getDiscoveryStatusBadgeTone(liveStatus.status);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = (event) => {
    if (!confirmNeeded) return;
    if (bypassConfirmRef.current) {
      bypassConfirmRef.current = false;
      return;
    }

    event.preventDefault();
    setConfirmOpen(true);
  };

  return (
    <Card>
      <CardBody>
        <section className="stack">
          <div className="split">
            <div className="stack" style={{ gap: 6 }}>
              <div style={{ fontSize: 12, letterSpacing: '0.05em', textTransform: 'uppercase', color: 'var(--muted-2)' }}>
                Discovery
              </div>

              <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                <Badge tone={badgeTone}>{liveStatus.status}</Badge>
                {latest?.scope ? <span className="hint">Scope: {latest.scope}</span> : null}
                {latestPreset ? <span className="hint">Preset: {getScanPresetLabel(latestPreset)}</span> : null}
                {latestTagLabel ? <span className="hint">Tags: {latestTagLabel}</span> : null}
              </div>

              <div className="hint" aria-live="polite">
                Last updated {formatLastUpdated(statusQuery.dataUpdatedAt)}
                {statusQuery.isFetching ? ' (refreshing…)'
                : ''}
              </div>

              {latest ? (
                <div className="hint">
                  Last run: {formatTimestamp(latest.started_at)}
                  {latest.completed_at ? ` → ${formatTimestamp(latest.completed_at)}` : null}
                  {latest.stats && typeof latest.stats === 'object' && 'stage' in latest.stats
                    ? ` (${String((latest.stats as Record<string, unknown>).stage)})`
                    : null}
                </div>
              ) : (
                <div className="hint">No discovery runs yet.</div>
              )}

              {progressLabel ? <div className="hint">{progressLabel}</div> : null}
              {latestSummary ? <div className="hint">{latestSummary}</div> : null}

              {inProgress ? (
                <div className="stack" style={{ gap: 6 }}>
                  <progress className="progressBar" aria-label="Discovery in progress" />
                </div>
              ) : null}

              {latest?.last_error ? <Alert tone="danger">Error: {latest.last_error}</Alert> : null}
            </div>

            <form
              ref={formRef}
              action={formAction}
              onSubmit={handleSubmit}
              className="stack"
              style={{ gap: 8, justifyItems: 'end' }}
            >
              {selectedTags.map((tag) => (
                <input key={tag} type="hidden" name="tags" value={tag} />
              ))}

              <Field>
                <Label htmlFor="preset">Preset</Label>
                <Select id="preset" name="preset" defaultValue={defaultPreset} disabled={readOnly}>
                  {SCAN_PRESET_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </Select>
                <Hint>
                  {SCAN_PRESET_OPTIONS.map((opt) => `${opt.label}: ${opt.description}`).join(' ')}
                </Hint>
              </Field>

              <details className="hint" style={{ marginTop: 4 }}>
                <summary style={{ cursor: 'pointer', fontWeight: 750 }}>Advanced: scan tags</summary>
                <div style={{ marginTop: 8 }}>
                  <Field>
                    <Label>Tags (optional)</Label>
                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                      {SCAN_TAG_OPTIONS.map((opt) => {
                        const active = selectedTags.includes(opt.value);
                        return (
                          <Button
                            key={opt.value}
                            type="button"
                            className={`btnPill${active ? ' btnPillActive' : ''}`}
                            aria-pressed={active}
                            onClick={() => {
                              setSelectedTags((prev) => {
                                if (prev.includes(opt.value)) {
                                  return prev.filter((t) => t !== opt.value);
                                }
                                return [...prev, opt.value];
                              });
                            }}
                          >
                            {opt.label}
                          </Button>
                        );
                      })}
                    </div>
                    <Hint>{SCAN_TAG_OPTIONS.map((opt) => `${opt.label}: ${opt.description}`).join(' ')}</Hint>
                  </Field>
                </div>
              </details>

              <Field>
                <Label htmlFor="scope">Scope (optional)</Label>
                <Input
                  id="scope"
                  name="scope"
                  placeholder="e.g. 10.0.0.0/24"
                  value={scopeValue}
                  onChange={(event) => setScopeValue(event.target.value)}
                  disabled={readOnly}
                />
                {scopeSuggestions.length > 0 ? (
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 6 }}>
                    {scopeSuggestions.slice(0, 8).map((suggestion) => {
                      const label = suggestion.interface ? `${suggestion.scope} (${suggestion.interface})` : suggestion.scope;
                      const active = scopeValue.trim() === suggestion.scope;
                      return (
                        <Button
                          key={`${suggestion.scope}:${suggestion.interface ?? ''}`}
                          type="button"
                          className={`btnPill${active ? ' btnPillActive' : ''}`}
                          onClick={() => setScopeValue(suggestion.scope)}
                        >
                          {label}
                        </Button>
                      );
                    })}
                  </div>
                ) : null}
                <Hint>
                  Leave blank to use the server default scope (`DISCOVERY_DEFAULT_SCOPE`). Without a scope, discovery relies on the current ARP cache and may return zero results.
                  Suggestions come from the scanner’s local interfaces—pick one you can route to. While a run is active, a new trigger will queue another run.
                </Hint>
              </Field>
              <Button type="submit" variant="primary" disabled={readOnly}>
                Trigger discovery
              </Button>
              {readOnly ? <span className="hint">Read-only access cannot trigger discoveries.</span> : null}
            </form>
          </div>

          {state.message ? (
            <Alert tone={state.status === 'error' ? 'danger' : 'success'}>{state.message}</Alert>
          ) : null}

          <ConfirmDialog
            open={confirmOpen}
            title="Discovery already in progress"
            description="A discovery run is currently queued or running. Triggering another discovery will queue an additional run. Do you want to continue?"
            confirmLabel="Queue another run"
            cancelLabel="Cancel"
            onCancel={() => setConfirmOpen(false)}
            onConfirm={() => {
              setConfirmOpen(false);
              bypassConfirmRef.current = true;
              formRef.current?.requestSubmit();
            }}
          />
        </section>
      </CardBody>
    </Card>
  );
}
