import { headers } from 'next/headers';
import { randomUUID } from 'crypto';
import { notFound, redirect } from 'next/navigation';
import Link from 'next/link';

import { Badge } from '@/app/_components/ui/Badge';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';

import { DeviceMetadataEditor } from '../DeviceMetadataEditor';
import { DeviceNameCandidatesPanel } from '../DeviceNameCandidatesPanel';
import { DeviceTagsPanel } from '../DeviceTagsPanel';
import type { Device, DeviceChangeFeed, DeviceFacts } from '../types';
import { formatTagLabel } from '../tags';
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

function FactsCard({ facts, device }: { facts: DeviceFacts; device: Device }) {
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
                <>
                  {sortedInterfaces.some((iface) => iface.mac) ? (
                    <div>
                      <div className="hint" style={{ marginBottom: 6 }}>No ARP-discovered MACs (common in Docker bridge mode). Interface MACs from SNMP:</div>
                      <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                        {sortedInterfaces.filter((iface) => iface.mac).map((iface) => (
                          <li key={`iface-mac-${iface.id}`}>
                            <code style={{ fontSize: 13 }}>{iface.mac}</code> <span className="hint">({iface.name ?? iface.id})</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : (
                    <div className="hint">No MAC addresses recorded yet.</div>
                  )}
                </>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
                  {sortedMacs.map((mac, idx) => (
                    <li key={`${mac.mac}-${mac.updated_at}`}>
                      <code style={{ fontSize: 13 }}>{mac.mac}</code>
                      {idx === 0 && device.mac_vendor ? <span className="vendorTag">{device.mac_vendor}</span> : null}
                      {' '}<span className="hint">({mac.interface_name ?? 'unknown interface'})</span>
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
                      {iface.mac ? <span className="hint"> · <code style={{ fontSize: 12 }}>{iface.mac}</code></span> : null}
                      {iface.pvid ? <span className="hint"> · PVID {iface.pvid}</span> : null}
                      {iface.mtu ? <span className="hint"> · MTU {iface.mtu}</span> : null}
                      {iface.speed_bps ? <span className="hint"> · {(iface.speed_bps / 1_000_000).toFixed(0)} Mbps</span> : null}
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
                <div className="snmpGrid">
                  <span className="snmpLabel">sysName</span>
                  <span>{facts.snmp.sys_name ?? '—'}</span>
                  {facts.snmp.os_family ? (
                    <><span className="snmpLabel">OS</span><span>{facts.snmp.os_family}{facts.snmp.os_version ? ` ${facts.snmp.os_version}` : ''}</span></>
                  ) : null}
                  {facts.snmp.sys_location ? (
                    <><span className="snmpLabel">Location</span><span>{facts.snmp.sys_location}</span></>
                  ) : null}
                  {facts.snmp.sys_descr ? (
                    <><span className="snmpLabel">sysDescr</span><span className="snmpSysDescr">{facts.snmp.sys_descr}</span></>
                  ) : null}
                  <span className="snmpLabel">Last polled</span>
                  <span className="hint">{formatDateTime(facts.snmp.last_success_at)}</span>
                  {facts.snmp.last_error ? (
                    <><span className="snmpLabel">Last error</span><span className="hint" style={{ color: 'var(--danger)' }}>{facts.snmp.last_error}</span></>
                  ) : null}
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
  const primaryMac = facts.macs.length > 0
    ? facts.macs[0].mac
    : facts.interfaces.find((i) => i.mac)?.mac ?? null;

  return (
    <section className="stack">
      <header className="deviceHeader">
        <div style={{ minWidth: 0 }}>
          <h1 className="pageTitle" style={{ overflowWrap: 'break-word', wordBreak: 'break-word' }}>{device.display_name ?? '(unnamed device)'}</h1>
          <p className="pageSubTitle">Inspect facts, edit metadata, and review the change timeline.</p>
        </div>
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', justifyContent: 'flex-end', alignItems: 'center', flexShrink: 0 }}>
          <Badge tone="info">ID {device.id.slice(0, 8)}</Badge>
          {device.primary_ip ? <Badge tone="info">IP {device.primary_ip}</Badge> : null}
          {primaryMac ? <Badge tone="info">MAC {primaryMac}</Badge> : null}
          {online ? <Badge tone="success">Online</Badge> : <Badge tone="neutral">Offline</Badge>}
          {changed ? <Badge tone="warning">Changed</Badge> : null}
          {device.os_guess ? (
            <Badge tone={
              device.os_guess_confidence === 'high' ? 'success'
              : device.os_guess_confidence === 'medium' ? 'warning'
              : 'neutral'
            }>
              {device.os_guess}
            </Badge>
          ) : null}
          {device.mac_vendor ? <Badge tone="info">{device.mac_vendor}</Badge> : null}
          {(device.tags ?? [])
            .filter((tag): tag is string => typeof tag === 'string' && tag.trim().length > 0)
            .slice(0, 6)
            .map((tag) => (
              <Badge key={tag} tone="neutral">
                {formatTagLabel(tag)}
              </Badge>
            ))}
        </div>
      </header>

      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
        <Link href="/devices" className="btnPill">
          ← Back to devices
        </Link>
      </div>

      <div className="deviceOverviewGrid">
        <Card>
          <CardBody>
            <div className="stack" style={{ gap: 10 }}>
              <p className="kicker">Identity</p>
              <div className="deviceOverviewMeta">
                <div className="deviceOverviewRow">
                  <span className="deviceOverviewLabel">Primary IP</span>
                  <span>{device.primary_ip ?? '—'}</span>
                </div>
                {primaryMac ? (
                  <div className="deviceOverviewRow">
                    <span className="deviceOverviewLabel">Primary MAC</span>
                    <span><code style={{ fontSize: 12 }}>{primaryMac}</code>{device.mac_vendor ? <span className="hint"> ({device.mac_vendor})</span> : null}</span>
                  </div>
                ) : null}
                {device.os_guess ? (
                  <div className="deviceOverviewRow">
                    <span className="deviceOverviewLabel">OS guess</span>
                    <span>
                      {device.os_guess}
                      {device.os_guess_confidence ? (
                        <span className={`confidenceDot confidence-${device.os_guess_confidence}`} title={`Confidence: ${device.os_guess_confidence}`} />
                      ) : null}
                    </span>
                  </div>
                ) : null}
                {facts.snmp?.os_family ? (
                  <div className="deviceOverviewRow">
                    <span className="deviceOverviewLabel">SNMP OS</span>
                    <span>{facts.snmp.os_family}{facts.snmp.os_version ? ` ${facts.snmp.os_version}` : ''}</span>
                  </div>
                ) : null}
              </div>
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody>
            <div className="stack" style={{ gap: 10 }}>
              <p className="kicker">Freshness</p>
              <div className="deviceOverviewMeta">
                <div className="deviceOverviewRow">
                  <span className="deviceOverviewLabel">Last seen</span>
                  <span>{formatDateTime(device.last_seen_at)}</span>
                </div>
                <div className="deviceOverviewRow">
                  <span className="deviceOverviewLabel">Last changed</span>
                  <span>{formatDateTime(device.last_change_at)}</span>
                </div>
                <div className="deviceOverviewRow">
                  <span className="deviceOverviewLabel">Facts refreshed</span>
                  <span>{formatDateTime(factsUpdatedAt)}</span>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      </div>

      <FactsCard facts={facts} device={device} />

      <details className="deviceCollapsible">
        <summary className="deviceCollapsibleSummary">
          <span className="kicker">Metadata</span>
          <span className="hint">Edit operator-owned fields and apply a friendly display name.</span>
        </summary>
        <Card>
          <CardBody className="stack" style={{ gap: 12 }}>
            <DeviceNameCandidatesPanel
              deviceId={device.id}
              currentDisplayName={device.display_name ?? null}
              readOnly={isReadOnly}
            />
            <DeviceMetadataEditor device={device} readOnly={isReadOnly} />
          </CardBody>
        </Card>
      </details>

      <details className="deviceCollapsible">
        <summary className="deviceCollapsibleSummary">
          <span className="kicker">Tags</span>
          <span className="hint">Manage classification tags for this device.</span>
        </summary>
        <DeviceTagsPanel deviceId={device.id} readOnly={isReadOnly} />
      </details>

      <details className="deviceCollapsible">
        <summary className="deviceCollapsibleSummary">
          <span className="kicker">History</span>
          <span className="hint">Timeline powered by the Phase 9 history endpoint.</span>
        </summary>
        <Card>
          <CardBody>
            <DeviceHistoryTimeline
              deviceId={device.id}
              initialEvents={history.events ?? []}
              initialCursor={history.cursor ?? null}
              limit={HISTORY_LIMIT}
            />
          </CardBody>
        </Card>
      </details>

      <div>
        <EmptyState title="Tip">
          Use this page to answer: “what is this device?”, “what changed?”, and “what’s the current truth?”.
        </EmptyState>
      </div>
    </section>
  );
}
