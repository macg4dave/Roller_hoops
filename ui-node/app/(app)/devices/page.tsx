import { DevicesDashboard } from './DevicesDashboard';
import type { DevicePage, DiscoveryStatus } from './types';
import { headers } from 'next/headers';
import { randomUUID } from 'crypto';
import { redirect } from 'next/navigation';
import { getSessionUser } from '../../../lib/auth/session';

const VALID_SORTS = ['created_desc', 'last_seen_desc', 'last_change_desc'] as const;
const VALID_STATUS = ['online', 'offline', 'changed'] as const;

type SortOption = (typeof VALID_SORTS)[number];
type StatusFilter = (typeof VALID_STATUS)[number];

type RawSearchParams = {
  [key: string]: string | string[] | undefined;
};

type DevicesSearchParams = {
  q?: string;
  status?: StatusFilter;
  sort?: SortOption;
  cursor?: string;
  limit?: string;
  seen_within_seconds?: string;
  changed_within_seconds?: string;
};

const DEFAULT_SORT: SortOption = 'created_desc';

function toSingleValue(value?: string | string[]): string | undefined {
  if (!value) {
    return undefined;
  }
  return Array.isArray(value) ? value[0] : value;
}

function normalizeSearchParams(params?: RawSearchParams): DevicesSearchParams {
  const normalized: DevicesSearchParams = {};
  const q = toSingleValue(params?.q)?.trim();
  if (q) {
    normalized.q = q;
  }

  const status = toSingleValue(params?.status);
  if (status && status !== 'all' && VALID_STATUS.includes(status as StatusFilter)) {
    normalized.status = status as StatusFilter;
  }

  const sort = toSingleValue(params?.sort);
  if (sort && VALID_SORTS.includes(sort as SortOption)) {
    normalized.sort = sort as SortOption;
  }

  const cursor = toSingleValue(params?.cursor);
  if (cursor) {
    normalized.cursor = cursor;
  }

  const limit = toSingleValue(params?.limit);
  if (limit) {
    normalized.limit = limit;
  }

  const seenWithin = toSingleValue(params?.seen_within_seconds);
  if (seenWithin) {
    normalized.seen_within_seconds = seenWithin;
  }

  const changedWithin = toSingleValue(params?.changed_within_seconds);
  if (changedWithin) {
    normalized.changed_within_seconds = changedWithin;
  }

  return normalized;
}

async function fetchDevicePage(params: DevicesSearchParams): Promise<DevicePage> {
  const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const query = new URLSearchParams();

  if (params.q) query.set('q', params.q);
  if (params.status) query.set('status', params.status);
  if (params.sort) query.set('sort', params.sort);
  if (params.cursor) query.set('cursor', params.cursor);
  if (params.limit) query.set('limit', params.limit);
  if (params.seen_within_seconds) query.set('seen_within_seconds', params.seen_within_seconds);
  if (params.changed_within_seconds) query.set('changed_within_seconds', params.changed_within_seconds);

  const url = `${base}/api/v1/devices${query.toString() ? `?${query.toString()}` : ''}`;
  const res = await fetch(url, {
    cache: 'no-store',
    headers: { 'X-Request-ID': reqId }
  });

  if (!res.ok) {
    throw new Error(`Failed to load devices: ${res.status}`);
  }

  return (await res.json()) as DevicePage;
}

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

export default async function DevicesPage({
  searchParams
}: {
  searchParams?: Promise<RawSearchParams>;
}) {
  const currentUser = await getSessionUser();
  if (!currentUser) {
    redirect('/auth/login');
  }

  const resolvedSearchParams = searchParams ? await searchParams : undefined;
  const normalizedParams = normalizeSearchParams(resolvedSearchParams);

  const [devicePage, discoveryStatus] = await Promise.all([
    fetchDevicePage(normalizedParams),
    fetchDiscoveryStatus()
  ]);

  const initialFilters = {
    search: normalizedParams.q ?? '',
    status: normalizedParams.status ?? 'all',
    sort: normalizedParams.sort ?? DEFAULT_SORT
  } as const;

  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Devices</h1>
        <p className="pageSubTitle">Triage devices, inspect facts, and edit metadata.</p>
      </header>

      <DevicesDashboard
        devicePage={devicePage}
        discoveryStatus={discoveryStatus}
        currentUser={currentUser}
        initialFilters={initialFilters}
      />
    </section>
  );
}
