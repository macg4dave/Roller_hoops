import { DevicesDashboard } from './DevicesDashboard';
import type { Device, DiscoveryStatus } from './types';
import { headers } from 'next/headers';
import { randomUUID } from 'crypto';

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

  return (await res.json()) as Device[];
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
  const [devices, discoveryStatus] = await Promise.all([fetchDevices(), fetchDiscoveryStatus()]);

  return (
    <main>
      <h1 style={{ fontSize: 28, marginBottom: 8 }}>Devices</h1>
      <p style={{ color: '#444' }}>
        CRUD backed by the Go API. The UI only talks to the API; no database access.
      </p>

      <DevicesDashboard devices={devices} discoveryStatus={discoveryStatus} />
    </main>
  );
}
