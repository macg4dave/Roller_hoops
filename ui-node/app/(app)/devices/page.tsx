import { DevicesDashboard } from './DevicesDashboard';
import type { Device, DiscoveryStatus } from './types';
import { headers } from 'next/headers';
import { randomUUID } from 'crypto';
import { redirect } from 'next/navigation';
import { getSessionUser } from '../../../lib/auth/session';

type DevicePage = { devices: Device[]; cursor?: string | null };

async function fetchDevices(): Promise<Device[]> {
  const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  const res = await fetch(`${base}/api/v1/devices`, {
    // Server-side fetch; avoid caching so dev changes show up immediately.
    cache: 'no-store',
    headers: { 'X-Request-ID': reqId }
  });

  if (!res.ok) {
    throw new Error(`Failed to load devices: ${res.status}`);
  }

  const page = (await res.json()) as DevicePage;
  return page.devices ?? [];
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

export default async function DevicesPage() {
  const currentUser = await getSessionUser();
  if (!currentUser) {
    redirect('/auth/login');
  }

  const [devices, discoveryStatus] = await Promise.all([fetchDevices(), fetchDiscoveryStatus()]);

  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Devices</h1>
        <p className="pageSubTitle">Triage devices, inspect facts, and edit metadata.</p>
      </header>

      <DevicesDashboard devices={devices} discoveryStatus={discoveryStatus} currentUser={currentUser} />
    </section>
  );
}
