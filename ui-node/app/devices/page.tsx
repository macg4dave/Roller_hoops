import { CreateDeviceForm } from './CreateDeviceForm';
import type { Device } from './types';

async function fetchDevices(): Promise<Device[]> {
  const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
  const res = await fetch(`${base}/api/v1/devices`, {
    // Server-side fetch; avoid caching so dev changes show up immediately.
    cache: 'no-store'
  });

  if (!res.ok) {
    throw new Error(`Failed to load devices: ${res.status}`);
  }

  return (await res.json()) as Device[];
}

export default async function DevicesPage() {
  const devices = await fetchDevices();

  return (
    <main>
      <h1 style={{ fontSize: 28, marginBottom: 8 }}>Devices</h1>
      <p style={{ color: '#444' }}>
        CRUD backed by the Go API. The UI only talks to the API; no database access.
      </p>

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
