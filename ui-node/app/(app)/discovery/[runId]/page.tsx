import Link from 'next/link';

import { Badge } from '@/app/_components/ui/Badge';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { Hint } from '@/app/_components/ui/Field';
import { Alert } from '@/app/_components/ui/Alert';
import { getDiscoveryStatusBadgeTone } from '../status';
import { DiscoveryRunLogViewer } from '../DiscoveryRunLogViewer';
import { fetchDiscoveryRun, fetchDiscoveryRunLogs } from '../api';
import type { DiscoveryRunLogPage } from '@/app/(app)/devices/types';

type Props = {
  params: Promise<{
    runId: string;
  }>;
};

function formatTimestamp(value?: string | null) {
  if (!value) return '—';
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(parsed);
}

function formatDurationMs(ms: number) {
  if (!Number.isFinite(ms) || ms < 0) return '—';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes <= 0) return `${seconds}s`;
  return `${minutes}m ${seconds}s`;
}

function computeDurationLabel(run: { started_at: string; completed_at?: string | null }) {
  const started = Date.parse(run.started_at);
  if (!Number.isFinite(started)) return null;
  const end = run.completed_at ? Date.parse(run.completed_at) : Date.now();
  if (!Number.isFinite(end)) return null;
  const ms = end - started;
  const prefix = run.completed_at ? 'Duration' : 'Elapsed';
  return `${prefix}: ${formatDurationMs(ms)}`;
}

function renderStats(run: { stats?: Record<string, unknown> | null }) {
  const entries = run.stats ? Object.entries(run.stats) : [];
  if (!entries.length) {
    return null;
  }
  return (
    <dl className="discoveryRunStats">
      {entries.map(([key, value]) => (
        <div key={key}>
          <dt>{key.replace(/_/g, ' ')}:</dt>
          <dd>{String(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

export default async function DiscoveryRunPage({ params }: Props) {
  const { runId } = await params;
  const run = await fetchDiscoveryRun(runId);
  const durationLabel = computeDurationLabel(run);
  let initialLogs: DiscoveryRunLogPage = { logs: [], cursor: null };
  let logError: string | null = null;
  try {
    initialLogs = await fetchDiscoveryRunLogs(run.id);
  } catch (error) {
    logError = error instanceof Error ? error.message : 'Unable to load run logs.';
  }

  return (
    <section className="stack discoveryRunDetail">
      <header>
        <p className="kicker">Discovery</p>
        <h1 className="pageTitle">Run {run.id}</h1>
        <p className="hint">
          Scope: {run.scope ?? 'default'}. Use this view to diagnose failures, review logs, and confirm completion status.
        </p>
      </header>

      <Card>
        <CardBody>
          <div className="split" style={{ alignItems: 'center' }}>
            <div>
              <div className="split" style={{ gap: 8, alignItems: 'center' }}>
                <h2 style={{ margin: 0, fontSize: 22 }}>{run.status}</h2>
                <Badge tone={getDiscoveryStatusBadgeTone(run.status)}>{run.status}</Badge>
              </div>
              <div className="hint">Started {formatTimestamp(run.started_at)}</div>
              <div className="hint">Completed {formatTimestamp(run.completed_at ?? null)}</div>
              {durationLabel ? <div className="hint">{durationLabel}</div> : null}
              <div className="hint">Cancellation: not supported (yet).</div>
            </div>
            <div className="hint" style={{ textAlign: 'right' }}>
              {run.last_error ? 'Failure detected' : 'Last run completed without error'}
            </div>
          </div>
          {run.last_error ? <Alert tone="danger">{run.last_error}</Alert> : null}
          {renderStats(run)}
        </CardBody>
      </Card>

      <DiscoveryRunLogViewer
        runId={run.id}
        initialLogs={initialLogs.logs ?? []}
        initialCursor={initialLogs.cursor ?? null}
        initialError={logError ?? undefined}
      />

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
        <Link href="/discovery" className="btn btnPrimary">
          Back to discovery
        </Link>
      </div>
    </section>
  );
}