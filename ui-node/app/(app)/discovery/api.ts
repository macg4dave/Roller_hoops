import { headers } from 'next/headers';
import { notFound } from 'next/navigation';
import { randomUUID } from 'crypto';

import type {
  DiscoveryRun,
  DiscoveryRunLogPage,
  DiscoveryRunPage,
  DiscoveryStatus
} from '@/app/(app)/devices/types';

const coreBaseUrl = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';

async function fetchWithRequestId(path: string, init?: RequestInit) {
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const headerInit: Record<string, string> = {
    'X-Request-ID': reqId,
    ...(init?.headers as Record<string, string>)
  };

  const response = await fetch(`${coreBaseUrl}${path}`, {
    cache: 'no-store',
    ...init,
    headers: headerInit
  });

  return response;
}

export async function fetchDiscoveryStatus(): Promise<DiscoveryStatus> {
  const response = await fetchWithRequestId('/api/v1/discovery/status');
  if (!response.ok) {
    throw new Error(`Failed to load discovery status (${response.status})`);
  }
  return response.json();
}

export async function fetchDiscoveryRuns(limit = 10, cursor?: string | null): Promise<DiscoveryRunPage> {
  const params = new URLSearchParams();
  params.set('limit', limit.toString());
  if (cursor) {
    params.set('cursor', cursor);
  }
  const response = await fetchWithRequestId(`/api/v1/discovery/runs?${params.toString()}`);
  if (!response.ok) {
    throw new Error(`Failed to load discovery runs (${response.status})`);
  }
  return response.json();
}

export async function fetchDiscoveryRun(runId: string): Promise<DiscoveryRun> {
  const encoded = encodeURIComponent(runId);
  const response = await fetchWithRequestId(`/api/v1/discovery/runs/${encoded}`);
  if (response.status === 404) {
    notFound();
  }
  if (!response.ok) {
    throw new Error(`Failed to load discovery run (${response.status})`);
  }
  return response.json();
}

export async function fetchDiscoveryRunLogs(runId: string, limit = 50, cursor?: string | null): Promise<DiscoveryRunLogPage> {
  const params = new URLSearchParams();
  params.set('limit', limit.toString());
  if (cursor) {
    params.set('cursor', cursor);
  }
  const encoded = encodeURIComponent(runId);
  const response = await fetchWithRequestId(`/api/v1/discovery/runs/${encoded}/logs?${params.toString()}`);
  if (!response.ok) {
    throw new Error(`Failed to load discovery run logs (${response.status})`);
  }
  return response.json();
}