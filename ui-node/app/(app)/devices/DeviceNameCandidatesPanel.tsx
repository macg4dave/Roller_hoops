'use client';

import { useEffect, useMemo, useState } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

import { Alert } from '@/app/_components/ui/Alert';
import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';

import type { DeviceNameCandidate } from './types';
import { updateDeviceDisplayName } from './actions';
import { initialDeviceDisplayNameState } from './state';

type Props = {
  deviceId: string;
  currentDisplayName?: string | null;
  readOnly?: boolean;
};

function normalizeName(name: string) {
  return name.trim().toLowerCase();
}

export function DeviceNameCandidatesPanel({ deviceId, currentDisplayName, readOnly = false }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(updateDeviceDisplayName, initialDeviceDisplayNameState());
  const [candidates, setCandidates] = useState<DeviceNameCandidate[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setLoadError(null);
      try {
        const res = await fetch(`/api/devices/${deviceId}/name-candidates`, { cache: 'no-store' });
        if (!res.ok) {
          throw new Error(`Request failed (${res.status})`);
        }
        const body = (await res.json()) as DeviceNameCandidate[];
        if (!cancelled) {
          setCandidates(body);
        }
      } catch (err) {
        if (!cancelled) {
          setCandidates(null);
          setLoadError(err instanceof Error ? err.message : 'Failed to load name candidates');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [deviceId, refreshKey]);

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
      setRefreshKey((v) => v + 1);
    }
  }, [router, state.status]);

  const current = useMemo(() => (typeof currentDisplayName === 'string' ? normalizeName(currentDisplayName) : ''), [currentDisplayName]);

  const displayCandidates = useMemo(() => {
    if (!candidates) return null;
    const deduped: DeviceNameCandidate[] = [];
    const seen = new Set<string>();
    for (const c of candidates) {
      const key = `${c.source}|${normalizeName(c.name)}|${c.address ?? ''}`;
      if (seen.has(key)) continue;
      seen.add(key);
      deduped.push(c);
    }
    return deduped.slice(0, 12);
  }, [candidates]);

  const statusTone = state.status === 'error' ? 'danger' : state.status === 'success' ? 'success' : 'info';

  return (
    <Card>
      <CardBody className="stack" style={{ gap: 10 }}>
        <div className="split" style={{ alignItems: 'baseline' }}>
          <div className="stack" style={{ gap: 6 }}>
            <p className="kicker">Name suggestions</p>
            <div style={{ fontSize: 14, fontWeight: 750 }}>
              {currentDisplayName?.trim() ? `Current: ${currentDisplayName}` : 'Current: (unnamed)'}
            </div>
          </div>

          <Button type="button" onClick={() => setRefreshKey((v) => v + 1)} disabled={loading || readOnly}>
            Refresh
          </Button>
        </div>

        {loading ? <div className="hint">Loading…</div> : null}
        {loadError ? <Alert tone="danger">{loadError}</Alert> : null}

        {displayCandidates && displayCandidates.length === 0 ? (
          <EmptyState title="No suggestions yet">
            Run discovery (and enable enrichment) to populate name candidates.
          </EmptyState>
        ) : null}

        {displayCandidates ? (
          <div className="nameCandidatesList">
            {displayCandidates.map((c) => {
              const isCurrent = current !== '' && normalizeName(c.name) === current;
              return (
                <form key={`${c.source}-${c.name}-${c.address ?? ''}-${c.observed_at}`} action={formAction} className="card">
                  <div className="cardPad nameCandidateRow">
                    <input type="hidden" name="device_id" value={deviceId} />
                    <input type="hidden" name="display_name" value={c.name} />

                    <div style={{ display: 'grid', gap: 4 }}>
                      <div className="nameCandidateTitle">
                        <span style={{ fontWeight: 800, fontSize: 14 }}>{c.name}</span>
                        <Badge>{c.source}</Badge>
                        {isCurrent ? <Badge tone="success">Current</Badge> : null}
                      </div>
                      <div className="hint">
                        {c.address ? `From ${c.address} · ` : null}
                        {new Date(c.observed_at).toLocaleString()}
                      </div>
                    </div>

                    <Button
                      type="submit"
                      variant={isCurrent ? 'default' : 'primary'}
                      disabled={isCurrent || readOnly}
                      title={
                        readOnly
                          ? 'Read-only users cannot update display names'
                          : isCurrent
                            ? 'Already the current display name'
                            : 'Apply as display name'
                      }
                    >
                      Use
                    </Button>
                  </div>
                </form>
              );
            })}
          </div>
        ) : null}

        {readOnly ? <Alert tone="warning">Read-only users cannot apply new display names.</Alert> : null}
        {state.message ? <Alert tone={statusTone}>{state.message}</Alert> : null}
      </CardBody>
    </Card>
  );
}

