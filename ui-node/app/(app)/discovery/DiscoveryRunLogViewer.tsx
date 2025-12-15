'use client';

import { useState } from 'react';

import { Alert } from '@/app/_components/ui/Alert';
import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Hint } from '@/app/_components/ui/Field';
import type { DiscoveryRunLogEntry, DiscoveryRunLogPage } from '@/app/(app)/devices/types';

type Props = {
  runId: string;
  initialLogs?: DiscoveryRunLogEntry[];
  initialCursor?: string | null;
  initialError?: string | null;
  limit?: number;
};

const DEFAULT_LIMIT = 40;

function formatTimestamp(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(parsed);
}

function levelTone(level: string) {
  switch (level.toLowerCase()) {
    case 'error':
      return 'danger';
    case 'warn':
    case 'warning':
      return 'warning';
    case 'info':
      return 'info';
    default:
      return 'neutral';
  }
}

export function DiscoveryRunLogViewer({
  runId,
  initialLogs = [],
  initialCursor = null,
  initialError,
  limit = DEFAULT_LIMIT
}: Props) {
  const [logs, setLogs] = useState(initialLogs);
  const [cursor, setCursor] = useState(initialCursor);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(initialError ?? null);
  const hasMore = Boolean(cursor);

  const fetchLogs = async (nextCursor?: string | null) => {
    const params = new URLSearchParams();
    params.set('limit', limit.toString());
    if (nextCursor) {
      params.set('cursor', nextCursor);
    }
    const response = await fetch(`/api/v1/discovery/runs/${encodeURIComponent(runId)}/logs?${params.toString()}`, {
      cache: 'no-store'
    });
    if (!response.ok) {
      throw new Error(`Failed to load run logs (${response.status})`);
    }
    return (await response.json()) as DiscoveryRunLogPage;
  };

  const handleLoadMore = async () => {
    if (!hasMore || loading) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const next = await fetchLogs(cursor);
      setLogs((previous) => [...previous, ...(next.logs ?? [])]);
      setCursor(next.cursor ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load more logs.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="discoveryLogSection">
      <CardBody>
        <div className="split" style={{ alignItems: 'center' }}>
          <div>
            <p className="kicker">Run logs</p>
            <h3 style={{ margin: 0 }}>Activity stream</h3>
          </div>
          <Hint>Every log line is emitted by the discovery worker; use this view to drill into failures.</Hint>
        </div>
        {error ? <Alert tone="danger">{error}</Alert> : null}
        {logs.length === 0 ? (
          <EmptyState title="No logs yet">
            Logs will appear as soon as the discovery worker starts processing the run.
          </EmptyState>
        ) : (
          <div className="discoveryLogList">
            {logs.map((log) => (
              <div key={`${log.id}-${log.created_at}`} className="runLogEntry">
                <div className="split" style={{ gap: 8 }}>
                  <Badge tone={levelTone(log.level)}>{log.level}</Badge>
                  <span className="hint" style={{ fontSize: 12 }}>
                    {formatTimestamp(log.created_at)}
                  </span>
                </div>
                <p>{log.message}</p>
              </div>
            ))}
          </div>
        )}
        <div className="discoveryLogFooter">
          <Button type="button" disabled={!hasMore || loading} onClick={handleLoadMore}>
            {loading ? 'Loading…' : 'Load more logs'}
          </Button>
          {!hasMore && logs.length > 0 ? <span className="hint">All logs loaded.</span> : null}
        </div>
      </CardBody>
    </Card>
  );
}