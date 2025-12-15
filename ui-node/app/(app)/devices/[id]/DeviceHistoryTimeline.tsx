'use client';

import { useCallback, useRef, useState } from 'react';

import { Button } from '@/app/_components/ui/Button';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Alert } from '@/app/_components/ui/Alert';

import type { DeviceChangeEvent, DeviceChangeFeed } from '../types';

const DEFAULT_LIMIT = 25;

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
});

function formatDateTime(value?: string | null) {
  if (!value) return '—';
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return '—';
  return dateTimeFormatter.format(parsed);
}

type Props = {
  deviceId: string;
  initialEvents: DeviceChangeEvent[];
  initialCursor?: string | null;
  limit?: number;
};

export function DeviceHistoryTimeline({ deviceId, initialEvents, initialCursor, limit = DEFAULT_LIMIT }: Props) {
  const [events, setEvents] = useState<DeviceChangeEvent[]>(() => initialEvents ?? []);
  const [cursor, setCursor] = useState<string | null>(() => initialCursor ?? null);
  const [status, setStatus] = useState<'idle' | 'loading' | 'error'>('idle');
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const loadMore = useCallback(async () => {
    if (!cursor || status === 'loading') {
      return;
    }

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setStatus('loading');
    setError(null);

    try {
      const params = new URLSearchParams();
      params.set('limit', String(limit));
      params.set('cursor', cursor);

      const res = await fetch(`/api/v1/devices/${deviceId}/history?${params.toString()}`, {
        cache: 'no-store',
        signal: controller.signal
      });

      if (controller.signal.aborted) return;
      if (!res.ok) {
        throw new Error(`History request failed: ${res.status}`);
      }

      const body = (await res.json()) as DeviceChangeFeed;
      if (controller.signal.aborted) return;

      setEvents((prev) => [...prev, ...(body.events ?? [])]);
      setCursor(body.cursor ?? null);
      setStatus('idle');
    } catch (err) {
      if (controller.signal.aborted) return;
      setError(err instanceof Error ? err.message : 'Unable to load more history.');
      setStatus('error');
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null;
      }
    }
  }, [cursor, deviceId, limit, status]);

  if (!events || events.length === 0) {
    return (
      <EmptyState title="No history yet">
        This device doesn’t have any recorded changes. Run discovery to generate observations.
      </EmptyState>
    );
  }

  return (
    <div className="stack" style={{ gap: 12 }}>
      <div className="split" style={{ alignItems: 'baseline' }}>
        <div className="hint">Newest first · Showing {events.length} events</div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {cursor ? (
            <Button type="button" onClick={loadMore} disabled={status === 'loading'}>
              {status === 'loading' ? 'Loading…' : 'Load more'}
            </Button>
          ) : (
            <div className="hint">All available events loaded.</div>
          )}
        </div>
      </div>

      {error ? <Alert tone="danger">{error}</Alert> : null}

      <div className="stack" style={{ gap: 10 }}>
        {events.map((event) => (
          <div key={event.event_id} className="card">
            <div className="cardPad" style={{ display: 'grid', gap: 6 }}>
              <div className="split" style={{ alignItems: 'baseline' }}>
                <div style={{ fontWeight: 800 }}>{event.summary}</div>
                <div className="hint">{formatDateTime(event.event_at)}</div>
              </div>
              <div className="hint">{event.kind}</div>
              {event.details && Object.keys(event.details).length > 0 ? (
                <div className="hint" style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace' }}>
                  {Object.entries(event.details)
                    .slice(0, 8)
                    .map(([key, value]) => (
                      <div key={key}>
                        {key}: {typeof value === 'string' ? value : JSON.stringify(value)}
                      </div>
                    ))}
                </div>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
