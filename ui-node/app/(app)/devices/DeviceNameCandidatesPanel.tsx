'use client';

import { useEffect, useMemo, useState } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

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

  return (
    <section style={{ border: '1px solid #f1f5f9', borderRadius: 8, padding: 12, display: 'grid', gap: 10, marginTop: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10, alignItems: 'baseline' }}>
        <div>
          <p style={{ margin: 0, fontSize: 12, letterSpacing: '0.08em', textTransform: 'uppercase', color: '#4b5563' }}>Name suggestions</p>
          <p style={{ margin: '6px 0 0', fontSize: 14, color: '#111827', fontWeight: 600 }}>
            {currentDisplayName?.trim() ? `Current: ${currentDisplayName}` : 'Current: (unnamed)'}
          </p>
        </div>
        <button
          type="button"
          onClick={() => setRefreshKey((v) => v + 1)}
          disabled={loading || readOnly}
          style={{
            borderRadius: 6,
            padding: '6px 10px',
            border: '1px solid #d1d5db',
            background: '#fff',
            cursor: loading || readOnly ? 'not-allowed' : 'pointer',
            fontWeight: 600,
            fontSize: 12
          }}
        >
          Refresh
        </button>
      </div>

      {loading ? <div style={{ color: '#6b7280', fontSize: 13 }}>Loading…</div> : null}
      {loadError ? <div style={{ color: '#b00020', fontSize: 13 }}>{loadError}</div> : null}

      {displayCandidates && displayCandidates.length === 0 ? (
        <div style={{ color: '#6b7280', fontSize: 13 }}>No suggestions yet. Run discovery (and enable enrichment) to populate candidates.</div>
      ) : null}

      {displayCandidates ? (
        <div style={{ display: 'grid', gap: 8 }}>
          {displayCandidates.map((c) => {
            const isCurrent = current !== '' && normalizeName(c.name) === current;
            return (
              <form
                key={`${c.source}-${c.name}-${c.address ?? ''}-${c.observed_at}`}
                action={formAction}
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'minmax(0, 1fr) auto',
                  gap: 10,
                  alignItems: 'center',
                  padding: '8px 10px',
                  borderRadius: 8,
                  border: '1px solid #e5e7eb',
                  background: '#fff'
                }}
              >
                <input type="hidden" name="device_id" value={deviceId} />
                <input type="hidden" name="display_name" value={c.name} />

                <div style={{ display: 'grid', gap: 4 }}>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'baseline', flexWrap: 'wrap' }}>
                    <span style={{ fontWeight: 700, fontSize: 14, color: '#111827' }}>{c.name}</span>
                    <span style={{ fontSize: 12, color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.08em' }}>{c.source}</span>
                  </div>
                  <div style={{ fontSize: 12, color: '#6b7280' }}>
                    {c.address ? `From ${c.address} · ` : null}
                    {new Date(c.observed_at).toLocaleString()}
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={isCurrent || readOnly}
                  style={{
                    background: isCurrent || readOnly ? '#e5e7eb' : '#111827',
                    color: isCurrent ? '#6b7280' : readOnly ? '#6b7280' : '#fff',
                    border: 'none',
                    borderRadius: 6,
                    padding: '8px 10px',
                    fontWeight: 700,
                    cursor: isCurrent || readOnly ? 'not-allowed' : 'pointer'
                  }}
                  title={
                    readOnly
                      ? 'Read-only users cannot update display names'
                      : isCurrent
                      ? 'Already the current display name'
                      : 'Apply as display name'
                  }
                >
                  Use
                </button>
              </form>
            );
          })}
        </div>
      ) : null}

      {readOnly ? (
        <p style={{ color: '#92400e', fontSize: 13, margin: 0 }}>Read-only users cannot apply new display names.</p>
      ) : null}

      {state.message ? (
        <p
          style={{
            margin: 0,
            color: state.status === 'error' ? '#b00020' : '#0f5132',
            background: state.status === 'error' ? '#f9d7da' : '#d1e7dd',
            borderRadius: 6,
            padding: '6px 10px',
            fontWeight: 600
          }}
        >
          {state.message}
        </p>
      ) : null}
    </section>
  );
}

