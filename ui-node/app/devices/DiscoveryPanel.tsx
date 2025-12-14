'use client';

import { useEffect } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

import type { DiscoveryStatus } from './types';
import { initialDiscoveryRunState } from './state';
import { triggerDiscovery } from './actions';

type Props = {
  status: DiscoveryStatus;
};

function formatTimestamp(ts?: string | null) {
  if (!ts) return 'n/a';
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return ts;
  }
  return d.toLocaleString();
}

function statusBadgeColor(status: string) {
  switch (status) {
    case 'running':
      return { bg: '#fef3c7', fg: '#92400e' };
    case 'succeeded':
      return { bg: '#dcfce7', fg: '#166534' };
    case 'failed':
      return { bg: '#fee2e2', fg: '#991b1b' };
    case 'queued':
      return { bg: '#e0e7ff', fg: '#3730a3' };
    default:
      return { bg: '#e5e7eb', fg: '#374151' };
  }
}

export function DiscoveryPanel({ status }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(triggerDiscovery, initialDiscoveryRunState());
  const latest = status.latest_run ?? undefined;

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
    }
  }, [state.status, router]);

  const colors = statusBadgeColor(status.status);

  return (
    <section
      style={{
        border: '1px solid #e0e0e0',
        borderRadius: 10,
        padding: 16,
        marginTop: 16,
        display: 'grid',
        gap: 10
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
        <div style={{ display: 'grid', gap: 6 }}>
          <div style={{ fontSize: 12, letterSpacing: '0.05em', textTransform: 'uppercase', color: '#4b5563' }}>
            Discovery
          </div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <span
              style={{
                background: colors.bg,
                color: colors.fg,
                padding: '4px 10px',
                borderRadius: 999,
                fontWeight: 700,
                fontSize: 13
              }}
            >
              {status.status}
            </span>
            {latest?.scope ? <span style={{ color: '#374151' }}>Scope: {latest.scope}</span> : null}
          </div>
          {latest ? (
            <div style={{ color: '#374151', fontSize: 14 }}>
              Last run: {formatTimestamp(latest.started_at)}
              {latest.completed_at ? ` → ${formatTimestamp(latest.completed_at)}` : null}
              {latest.stats && typeof latest.stats === 'object' && 'stage' in latest.stats
                ? ` (${String((latest.stats as Record<string, unknown>).stage)})`
                : null}
            </div>
          ) : (
            <div style={{ color: '#6b7280', fontSize: 14 }}>No discovery runs yet.</div>
          )}
          {latest?.last_error ? (
            <div style={{ color: '#991b1b', fontWeight: 600, fontSize: 14 }}>Error: {latest.last_error}</div>
          ) : null}
        </div>

        <form action={formAction} style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            name="scope"
            placeholder="optional scope (e.g. 10.0.0.0/24)"
            style={{
              padding: '10px 12px',
              borderRadius: 8,
              border: '1px solid #d1d5db',
              minWidth: 220
            }}
          />
          <button
            type="submit"
            style={{
              background: '#111827',
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              padding: '10px 14px',
              fontWeight: 700,
              cursor: 'pointer'
            }}
          >
            Trigger discovery
          </button>
        </form>
      </div>

      {state.message ? (
        <p
          style={{
            margin: 0,
            color: state.status === 'error' ? '#b00020' : '#0f5132',
            background: state.status === 'error' ? '#f9d7da' : '#d1e7dd',
            borderRadius: 6,
            padding: '8px 10px',
            fontWeight: 600
          }}
        >
          {state.message}
        </p>
      ) : null}
    </section>
  );
}
