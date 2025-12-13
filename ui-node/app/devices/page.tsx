type Device = {
  id: string;
  display_name?: string;
};

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
      <h1>Devices</h1>
      <p>Fetched from core-go via internal Docker network.</p>

      {devices.length === 0 ? (
        <p>No devices yet.</p>
      ) : (
        <ul>
          {devices.map((d) => (
            <li key={d.id}>
              <strong>{d.display_name ?? '(unnamed)'}</strong> — <code>{d.id}</code>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
