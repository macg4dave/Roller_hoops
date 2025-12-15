'use client';

import { useEffect, useState } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';

import type { DiscoveryStatus } from './types';
import { initialDiscoveryRunState } from './state';
import { triggerDiscovery } from './actions';
import { api } from '../../../lib/api-client';
import { Card, CardBody } from '../../_components/ui/Card';
import { Field, Hint, Label } from '../../_components/ui/Field';
import { Input } from '../../_components/ui/Inputs';
import { Button } from '../../_components/ui/Button';
import { Badge } from '../../_components/ui/Badge';
import { Alert } from '../../_components/ui/Alert';

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

function statusBadgeColor(status: string) {
  switch (status) {
    case 'running':
      return 'warning' as const;
    case 'succeeded':
      return 'success' as const;
    case 'failed':
      return 'danger' as const;
    case 'queued':
      return 'info' as const;
    default:
      return 'neutral' as const;
  }
}

export function DiscoveryPanel({ status, readOnly = false }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(triggerDiscovery, initialDiscoveryRunState());
  const [liveStatus, setLiveStatus] = useState<DiscoveryStatus>(status);

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
    refetchInterval: 10_000
  });

  const latest = statusQuery.data.latest_run ?? undefined;

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
    }
  }, [state.status, router]);

  useEffect(() => {
    setLiveStatus(statusQuery.data);
  }, [statusQuery.data]);

  const badgeTone = statusBadgeColor(liveStatus.status);

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

              {latest?.last_error ? <Alert tone="danger">Error: {latest.last_error}</Alert> : null}
            </div>

            <form action={formAction} className="stack" style={{ gap: 8, justifyItems: 'end' }}>
              <Field>
                <Label htmlFor="scope">Scope (optional)</Label>
                <Input
                  id="scope"
                  name="scope"
                  placeholder="e.g. 10.0.0.0/24"
                  disabled={readOnly}
                />
                <Hint>Leave blank to run with the default scope.</Hint>
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
        </section>
      </CardBody>
    </Card>
  );
}
