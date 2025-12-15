import { headers } from 'next/headers';
import { randomUUID } from 'crypto';
import { notFound, redirect } from 'next/navigation';
import Link from 'next/link';

import { Badge } from '@/app/_components/ui/Badge';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';

import { DeviceMetadataEditor } from '../DeviceMetadataEditor';
import { DeviceNameCandidatesPanel } from '../DeviceNameCandidatesPanel';
import type { Device, DeviceChangeFeed, DeviceFacts } from '../types';
import { getSessionUser } from '../../../../lib/auth/session';
import { DeviceHistoryTimeline } from './DeviceHistoryTimeline';

const HISTORY_LIMIT = 25;

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
});

function formatDateTime(value?: string | null) {
  if (!value) return '—';
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return '—';
  return dateTimeFormatter.format(parsed);
}

function isWithinSeconds(timestamp: string | null | undefined, seconds: number) {
  if (!timestamp) return false;
  const ts = Date.parse(timestamp);
  if (!Number.isFinite(ts)) return false;
  return Date.now() - ts <= seconds * 1000;
}

async function fetchFromCore(path: string): Promise<Response> {
  const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
  const reqId = (await headers()).get('x-request-id') ?? randomUUID();
  return fetch(`${base}${path}`, {
    cache: 'no-store',
    headers: {
      Accept: 'application/json',
      'X-Request-ID': reqId
    }
  });
}

async function fetchDevice(deviceId: string): Promise<Device> {
  const res = await fetchFromCore(`/api/v1/devices/${deviceId}`);
  if (res.status === 404) {
    notFound();
  }
  if (!res.ok) {
    throw new Error(`Failed to load device: ${res.status}`);
  }
  return (await res.json()) as Device;
}

async function fetchFacts(deviceId: string): Promise<DeviceFacts> {
  const res = await fetchFromCore(`/api/v1/devices/${deviceId}/facts`);
  if (!res.ok) {
    throw new Error(`Failed to load facts: ${res.status}`);
  }
  return (await res.json()) as DeviceFacts;
}

async function fetchHistory(deviceId: string): Promise<DeviceChangeFeed> {
  const params = new URLSearchParams();
  params.set('limit', String(HISTORY_LIMIT));

  const res = await fetchFromCore(`/api/v1/devices/${deviceId}/history?${params.toString()}`);
  if (!res.ok) {
    throw new Error(`Failed to load history: ${res.status}`);
  }
  return (await res.json()) as DeviceChangeFeed;
}

function FactsCard({ facts }: { facts: DeviceFacts }) {
  const sortedIps = [...facts.ips].sort((a, b) => a.ip.localeCompare(b.ip));
  const sortedMacs = [...facts.macs].sort((a, b) => a.mac.localeCompare(b.mac));
  const sortedInterfaces = [...facts.interfaces].sort((a, b) => (a.name ?? '').localeCompare(b.name ?? ''));
  const sortedServices = [...facts.services].sort((a, b) => {
    const portA = a.port ?? 0;
    const portB = b.port ?? 0;
    if (portA !== portB) return portA - portB;
    return (a.protocol ?? '').localeCompare(b.protocol ?? '');
  });

  return (
    <Card>
      <CardBody>
        <div className="stack" style={{ gap: 12 }}>
          <div>
            <p className="kicker">Facts</p>
            <p className="hint">Current truth from discovery and enrichment (IPs, MACs, interfaces, services, SNMP, links).</p>
          </div>

          <div className="stack" style={{ gap: 14 }}>
            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>IP addresses</h3>
              {sortedIps.length === 0 ? (
                <div className="hint">No IP addresses recorded yet.</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {sortedIps.map((ip) => (
                    <li key={`${ip.ip}-${ip.updated_at}`}>
                      <strong>{ip.ip}</strong> <span className="hint">({ip.interface_name ?? 'unknown interface'})</span>
                      <div className="hint">Updated {formatDateTime(ip.updated_at)}</div>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>MAC addresses</h3>
              {sortedMacs.length === 0 ? (
                <div className="hint">No MAC addresses recorded yet.</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {sortedMacs.map((mac) => (
                    <li key={`${mac.mac}-${mac.updated_at}`}>
                      <strong>{mac.mac}</strong> <span className="hint">({mac.interface_name ?? 'unknown interface'})</span>
                      <div className="hint">Updated {formatDateTime(mac.updated_at)}</div>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>Interfaces</h3>
              {sortedInterfaces.length === 0 ? (
                <div className="hint">No interfaces recorded yet.</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {sortedInterfaces.map((iface) => (
                    <li key={`${iface.id}-${iface.updated_at}`}>
                      <strong>{iface.name ?? iface.id}</strong>
                      {iface.pvid ? <span className="hint"> · PVID {iface.pvid}</span> : null}
                      {iface.mtu ? <span className="hint"> · MTU {iface.mtu}</span> : null}
                      <div className="hint">Updated {formatDateTime(iface.updated_at)}</div>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>Services</h3>
              {sortedServices.length === 0 ? (
                <div className="hint">No services recorded yet.</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {sortedServices.map((svc, idx) => (
                    <li key={`${svc.protocol ?? 'unknown'}-${svc.port ?? 'none'}-${svc.name ?? ''}-${idx}`}>
                      <strong>
                        {(svc.protocol ?? 'tcp').toUpperCase()} {svc.port ?? '—'}
                      </strong>
                      {svc.name ? <span className="hint"> · {svc.name}</span> : null}
                      {svc.state ? <span className="hint"> · {svc.state}</span> : null}
                      {svc.source ? <div className="hint">Source: {svc.source}</div> : null}
                      <div className="hint">Observed {formatDateTime(svc.observed_at)}</div>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>SNMP snapshot</h3>
              {facts.snmp ? (
                <div className="stack" style={{ gap: 4 }}>
                  {facts.snmp.sys_name ? <div>sysName: {facts.snmp.sys_name}</div> : <div className="hint">sysName: —</div>}
                  {facts.snmp.sys_descr ? <div>sysDescr: {facts.snmp.sys_descr}</div> : null}
                  {facts.snmp.sys_location ? <div>sysLocation: {facts.snmp.sys_location}</div> : null}
                  {facts.snmp.last_success_at ? <div className="hint">Last success: {formatDateTime(facts.snmp.last_success_at)}</div> : null}
                  {facts.snmp.last_error ? <div className="hint">Last error: {facts.snmp.last_error}</div> : null}
                  <div className="hint">Updated {formatDateTime(facts.snmp.updated_at)}</div>
                </div>
              ) : (
                <div className="hint">No SNMP snapshot recorded.</div>
              )}
            </div>

            <div>
              <h3 style={{ margin: '0 0 6px', fontSize: 16 }}>Links</h3>
              {facts.links.length === 0 ? (
                <div className="hint">No adjacency links recorded.</div>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {facts.links.map((link) => (
                    <li key={link.id}>
                      <strong>{link.source}</strong> <span className="hint">· peer {link.peer_device_id.slice(0, 8)}</span>
                      {link.link_type ? <span className="hint"> · {link.link_type}</span> : null}
                      <div className="hint">Updated {formatDateTime(link.updated_at)}</div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      </CardBody>
    </Card>
  );
}

export default async function DeviceDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const currentUser = await getSessionUser();
  if (!currentUser) {
    redirect('/auth/login');
  }

  const { id } = await params;

  const [device, facts, history] = await Promise.all([fetchDevice(id), fetchFacts(id), fetchHistory(id)]);

  const online = isWithinSeconds(device.last_seen_at, 3600);
  const changed = isWithinSeconds(device.last_change_at, 86400);
  const isReadOnly = currentUser.role === 'read-only';

  const factsUpdatedAt = facts.snmp?.updated_at ?? device.last_change_at ?? null;

  return (
    <section className="stack">
      <header className="split" style={{ alignItems: 'baseline' }}>
        <div>
          <h1 className="pageTitle">{device.display_name ?? '(unnamed device)'}</h1>
          <p className="pageSubTitle">Inspect facts, edit metadata, and review the change timeline.</p>
        </div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
          <Badge tone="info">ID {device.id.slice(0, 8)}</Badge>
          {device.primary_ip ? <Badge tone="info">IP {device.primary_ip}</Badge> : null}
          {online ? <Badge tone="success">Online</Badge> : <Badge tone="neutral">Offline</Badge>}
          {changed ? <Badge tone="warning">Changed</Badge> : null}
        </div>
      </header>

      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
        <Link href="/devices" className="btnPill">
          ← Back to devices
        </Link>
      </div>

      <Card>
        <CardBody>
          <div className="stack" style={{ gap: 10 }}>
            <div>
              <p className="kicker">Overview</p>
              <p className="hint">High-level identity and freshness indicators.</p>
            </div>

            <div style={{ display: 'grid', gap: 4, fontSize: 13, color: 'var(--muted)' }}>
              <div>Primary IP: {device.primary_ip ?? '—'}</div>
              <div>Last seen: {formatDateTime(device.last_seen_at)}</div>
              <div>Last changed: {formatDateTime(device.last_change_at)}</div>
              <div>Facts refreshed: {formatDateTime(factsUpdatedAt)}</div>
            </div>
          </div>
        </CardBody>
      </Card>

      <FactsCard facts={facts} />

      <Card>
        <CardBody className="stack" style={{ gap: 12 }}>
          <div>
            <p className="kicker">Metadata</p>
            <p className="hint">Edit operator-owned fields and apply a friendly display name.</p>
          </div>
          <DeviceNameCandidatesPanel
            deviceId={device.id}
            currentDisplayName={device.display_name ?? null}
            readOnly={isReadOnly}
          />
          <DeviceMetadataEditor device={device} readOnly={isReadOnly} />
        </CardBody>
      </Card>

      <Card>
        <CardBody>
          <div className="stack" style={{ gap: 12 }}>
            <div>
              <p className="kicker">History</p>
              <p className="hint">Timeline powered by the Phase 9 history endpoint (no UI-side diffing).</p>
            </div>

            <DeviceHistoryTimeline
              deviceId={device.id}
              initialEvents={history.events ?? []}
              initialCursor={history.cursor ?? null}
              limit={HISTORY_LIMIT}
            />
          </div>
        </CardBody>
      </Card>

      <div>
        <EmptyState title="Tip">
          Use this page to answer: “what is this device?”, “what changed?”, and “what’s the current truth?”.
        </EmptyState>
      </div>
    </section>
  );
}
