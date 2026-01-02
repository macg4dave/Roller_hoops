'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { api } from '@/lib/api-client';
import { Alert } from '@/app/_components/ui/Alert';
import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Hint } from '@/app/_components/ui/Field';
import { getDiscoveryStatusBadgeTone } from './status';
import type { DiscoveryRun, DiscoveryRunPage } from '@/app/(app)/devices/types';
import { getScanPresetLabel } from './presets';

const PAGE_LIMIT = 8;

function getApiErrorMessage(error: unknown): string | undefined {
  if (!error || typeof error !== 'object') {
    return undefined;
  }

  const record = error as Record<string, unknown>;
  if (typeof record.message === 'string') {
    return record.message;
  }
  const nested = record.error;
  if (nested && typeof nested === 'object' && typeof (nested as Record<string, unknown>).message === 'string') {
    return (nested as Record<string, unknown>).message as string;
  }

  return undefined;
}

async function loadDiscoveryRuns(cursor?: string | null, limit = PAGE_LIMIT) {
  const response = await api.GET('/v1/discovery/runs', {
    query: {
      limit,
      cursor: cursor ?? undefined
    }
  });
  if (response.error) {
    throw new Error(getApiErrorMessage(response.error) ?? 'Failed to load discovery runs.');
  }
  return response.data ?? { runs: [], cursor: null };
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '—';
  }
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(parsed);
}

function formatLastUpdated(updatedAtMs: number) {
  if (!Number.isFinite(updatedAtMs) || updatedAtMs <= 0) return '—';
  const deltaMs = Date.now() - updatedAtMs;
  if (deltaMs < 2000) return 'just now';
  if (deltaMs < 60_000) return `${Math.round(deltaMs / 1000)}s ago`;
  if (deltaMs < 60 * 60_000) return `${Math.round(deltaMs / 60_000)}m ago`;
  return new Date(updatedAtMs).toLocaleString();
}

type Props = {
  initialPage?: DiscoveryRunPage | null;
  limit?: number;
  errorMessage?: string;
};

export function DiscoveryRunList({ initialPage, limit = PAGE_LIMIT, errorMessage }: Props) {
  const router = useRouter();
  const { data, isError, error, isFetching, dataUpdatedAt } = useQuery<DiscoveryRunPage, Error>({
    queryKey: ['discovery-runs-page', limit],
    queryFn: () => loadDiscoveryRuns(null, limit),
    ...(initialPage ? { initialData: initialPage } : {}),
    refetchInterval: (query) => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        return false;
      }
      return query.state.status === 'error' ? 30_000 : 10_000;
    },
    refetchIntervalInBackground: false,
    placeholderData: keepPreviousData
  });
  const [pages, setPages] = useState<DiscoveryRunPage[]>(() => (initialPage ? [initialPage] : []));
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState<string | null>(null);

  useEffect(() => {
    if (!data) {
      return;
    }
    setPages((previous) => {
      if (previous.length === 0) {
        return [data];
      }
      const next = [...previous];
      next[0] = data;
      return next;
    });
  }, [data]);

  const allRuns = useMemo(() => pages.flatMap((page) => page.runs ?? []), [pages]);
  const hasMore = Boolean(pages.at(-1)?.cursor);
  const nextCursor = pages.at(-1)?.cursor ?? null;

  const handleLoadMore = async () => {
    if (!hasMore || loadingMore) {
      return;
    }
    setLoadingMore(true);
    setLoadMoreError(null);
    try {
      const nextPage = await loadDiscoveryRuns(nextCursor, limit);
      setPages((prev) => [...prev, nextPage]);
    } catch (err) {
      setLoadMoreError(err instanceof Error ? err.message : 'Unable to load older runs.');
    } finally {
      setLoadingMore(false);
    }
  };

  const statusErrorMessage = error instanceof Error ? error.message : undefined;

  const renderStats = (run: DiscoveryRun) => {
    const entries = run.stats ? Object.entries(run.stats) : [];
    const trimmed = entries
      .filter(([key, value]) => key !== 'preset' && key !== 'stage' && value !== undefined && value !== null)
      .slice(0, 3);
    if (!trimmed.length) {
      return null;
    }
    return (
      <div className="discoveryRunStats">
        {trimmed.map(([key, value]) => (
          <div key={key}>
            <strong>{key.replace(/_/g, ' ')}:</strong> {String(value)}
          </div>
        ))}
      </div>
    );
  };

  return (
    <Card className="discoveryRunSection">
      <CardBody>
        <div className="split" style={{ alignItems: 'center' }}>
          <div>
            <p className="kicker">Discovery runs</p>
            <h2 style={{ margin: '6px 0' }}>Run history</h2>
          </div>
          <Hint aria-live="polite">
            Last updated {formatLastUpdated(dataUpdatedAt)}
            {isFetching ? ' (refreshing…)'
            : ''}
          </Hint>
        </div>
        {errorMessage ? <Alert tone="danger">{errorMessage}</Alert> : null}
        {isError ? <Alert tone="danger">{statusErrorMessage ?? 'Unable to refresh run list.'}</Alert> : null}
        {allRuns.length === 0 && !isFetching ? (
          <EmptyState title="No discovery runs yet">
            Trigger a sweep above to populate this list and inspect the resulting logs.
          </EmptyState>
        ) : (
          <div className="discoveryRunGrid">
            {allRuns.map((run) => (
              <Card key={run.id} className="discoveryRunCard">
                <CardBody>
                  <div className="split" style={{ alignItems: 'center' }}>
                    <div>
                      <div className="split" style={{ gap: 8, alignItems: 'center' }}>
                        <span style={{ fontWeight: 700 }}>{run.id.slice(0, 8)}</span>
                        <Badge tone={getDiscoveryStatusBadgeTone(run.status)}>{run.status}</Badge>
                      </div>
                      <div className="hint" style={{ marginTop: 4 }}>
                        Scope: {run.scope ?? 'default'} • Preset: {getScanPresetLabel(run.stats?.preset)}
                      </div>
                      <div className="hint" style={{ marginTop: 2 }}>
                        Started {formatTimestamp(run.started_at)}
                        {run.completed_at ? ` → ${formatTimestamp(run.completed_at)}` : ''}
                      </div>
                    </div>
                    <Button type="button" onClick={() => router.push(`/discovery/${run.id}`)}>
                      View details
                    </Button>
                  </div>
                  {run.last_error ? <Alert tone="danger">{run.last_error}</Alert> : null}
                  {renderStats(run)}
                </CardBody>
              </Card>
            ))}
          </div>
        )}
        <div className="discoveryRunFooter">
          <Button type="button" disabled={!hasMore || loadingMore} onClick={handleLoadMore}>
            {loadingMore ? 'Loading…' : 'Load more runs'}
          </Button>
          {loadMoreError ? <span className="hint">{loadMoreError}</span> : null}
        </div>
      </CardBody>
    </Card>
  );
}
