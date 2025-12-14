import { CreateDeviceForm } from './CreateDeviceForm';
import type { Device, DiscoveryStatus } from './types';
import { DiscoveryPanel } from './DiscoveryPanel';
import { DeviceMetadataEditor } from './DeviceMetadataEditor';
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

      <DiscoveryPanel status={discoveryStatus} />
      <CreateDeviceForm />

      {devices.length === 0 ? (
        <p style={{ marginTop: 16 }}>No devices yet. Create the first device above.</p>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, marginTop: 24, display: 'grid', gap: 12 }}>
          {devices.map((d) => (
            <li
              key={d.id}
              style={{
                border: '1px solid #e0e0e0',
                borderRadius: 8,
                padding: 12,
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}
            >
              <div>
                <div style={{ fontWeight: 700 }}>{d.display_name ?? '(unnamed)'}</div>
                <div style={{ color: '#555', fontSize: 14 }}>
                  <code>{d.id}</code>
                </div>
                <div style={{ color: '#333', fontSize: 14, marginTop: 6, display: 'grid', gap: 4 }}>
                  {d.metadata?.owner ? (
                    <div>
                      <strong>Owner:</strong> {d.metadata.owner}
                    </div>
                  ) : null}
                  {d.metadata?.location ? (
                    <div>
                      <strong>Location:</strong> {d.metadata.location}
                    </div>
                  ) : null}
                  {d.metadata?.notes ? (
                    <div style={{ color: '#444' }}>
                      <strong>Notes:</strong> {d.metadata.notes}
                    </div>
                  ) : null}
                  {!d.metadata && <div style={{ color: '#666' }}>No metadata yet.</div>}
                </div>
                <DeviceMetadataEditor device={d} />
              </div>
              <span
                style={{
                  background: '#eef2ff',
                  color: '#4338ca',
                  padding: '4px 8px',
                  borderRadius: 999,
                  fontSize: 12,
                  fontWeight: 700
                }}
              >
                device
              </span>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
