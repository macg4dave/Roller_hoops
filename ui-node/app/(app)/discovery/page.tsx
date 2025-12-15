import { headers } from 'next/headers';
import { randomUUID } from 'crypto';

import { DiscoveryPanel } from '../devices/DiscoveryPanel';
import type { DiscoveryStatus } from '../devices/types';
import { getSessionUser } from '../../../lib/auth/session';
import { Card, CardBody } from '../../_components/ui/Card';
import { EmptyState } from '../../_components/ui/EmptyState';

async function fetchDiscoveryStatus(): Promise<DiscoveryStatus> {
  const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const res = await fetch(`${base}/api/v1/discovery/status`, {
    cache: 'no-store',
    headers: { 'X-Request-ID': reqId }
  });

  if (!res.ok) {
    throw new Error(`Failed to load discovery status: ${res.status}`);
  }

  return (await res.json()) as DiscoveryStatus;
}

export default async function DiscoveryPage() {
  const currentUser = await getSessionUser();
  const discoveryStatus = await fetchDiscoveryStatus();
  const readOnly = currentUser?.role === 'read-only';

  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Discovery</h1>
        <p className="pageSubTitle">Run discovery, monitor progress, and debug failures.</p>
      </header>

      <DiscoveryPanel status={discoveryStatus} readOnly={readOnly} />

      <Card>
        <CardBody>
          <EmptyState title="Discovery run history is next">
            Phase 12 adds a run list + run detail + logs view here (backed by the existing discovery runs APIs).
          </EmptyState>
        </CardBody>
      </Card>
    </section>
  );
}
