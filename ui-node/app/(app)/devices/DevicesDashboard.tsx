'use client';

import { ChangeEvent, FormEvent, MouseEvent, useCallback, useEffect, useRef, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Field, Label } from '@/app/_components/ui/Field';
import { Input, Select } from '@/app/_components/ui/Inputs';

import { CreateDeviceForm } from './CreateDeviceForm';
import { DeviceNameCandidatesPanel } from './DeviceNameCandidatesPanel';
import { DeviceMetadataEditor } from './DeviceMetadataEditor';
import { DiscoveryPanel } from './DiscoveryPanel';
import { ImportExportPanel } from './ImportExportPanel';
import type {
  DeviceChangeEvent,
  DeviceChangeFeed,
  DeviceFacts,
  DevicePage,
  DiscoveryStatus
} from './types';

type SessionUser = {
  username: string;
  role: string;
};

const STATUS_OPTIONS = [
  { id: 'all', label: 'All devices' },
  { id: 'online', label: 'Online' },
  { id: 'offline', label: 'Offline' },
  { id: 'changed', label: 'Recently changed' }
] as const;

const SORT_OPTIONS = [
  { id: 'created_desc', label: 'Newest created' },
  { id: 'last_seen_desc', label: 'Recently seen' },
  { id: 'last_change_desc', label: 'Recently changed' }
] as const;

type StatusFilter = (typeof STATUS_OPTIONS)[number]['id'];
type SortOption = (typeof SORT_OPTIONS)[number]['id'];

type FilterState = {
  search: string;
  status: StatusFilter;
  sort: SortOption;
};

type AsyncStatus = 'idle' | 'loading' | 'success' | 'error';

const HISTORY_PAGE_LIMIT = 20;
const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
});

function formatDateTime(value?: string | null) {
  if (!value) {
    return '—';
  }
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return '—';
  }
  return dateTimeFormatter.format(parsed);
}

type Props = {
  devicePage: DevicePage;
  discoveryStatus: DiscoveryStatus;
  currentUser: SessionUser;
  initialFilters: FilterState;
};

export function DevicesDashboard({ devicePage, discoveryStatus, currentUser, initialFilters }: Props) {
  const router = useRouter();
  const pathname = usePathname() ?? '/devices';
  const searchParams = useSearchParams();

  const devices = devicePage.devices;
  const [selectedId, setSelectedId] = useState<string | undefined>(() => devices[0]?.id);
  const [searchInput, setSearchInput] = useState(initialFilters.search);
  const isReadOnly = currentUser.role === 'read-only';

  const currentSearch = searchParams?.get('q') ?? initialFilters.search;
  const rawStatus = searchParams?.get('status') ?? initialFilters.status;
  const currentStatus: StatusFilter = STATUS_OPTIONS.some((option) => option.id === rawStatus)
    ? (rawStatus as StatusFilter)
    : 'all';
  const rawSort = searchParams?.get('sort') ?? initialFilters.sort;
  const currentSort: SortOption = SORT_OPTIONS.some((option) => option.id === rawSort)
    ? (rawSort as SortOption)
    : initialFilters.sort;

  useEffect(() => {
    setSearchInput(currentSearch);
  }, [currentSearch]);

  useEffect(() => {
    setSelectedId((current) => {
      if (!devices.length) {
        return undefined;
      }
      if (current && devices.some((device) => device.id === current)) {
        return current;
      }
      return devices[0].id;
    });
  }, [devices]);

  const updateParams = (changes: Record<string, string | undefined | null>) => {
    const nextParams = new URLSearchParams(searchParams?.toString() ?? '');
    Object.entries(changes).forEach(([key, value]) => {
      if (value === undefined || value === null || value === '') {
        nextParams.delete(key);
      } else {
        nextParams.set(key, value);
      }
    });
    const queryString = nextParams.toString();
    router.push(`${pathname}${queryString ? `?${queryString}` : ''}`);
  };

  const handleSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    updateParams({ q: searchInput.trim() || undefined, cursor: undefined });
  };

  const handleStatusToggle = (status: StatusFilter) => {
    const normalized = status === 'all' ? undefined : status;
    updateParams({ status: normalized, cursor: undefined });
  };

  const handleSortChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextSort = event.currentTarget.value as SortOption;
    if (SORT_OPTIONS.some((option) => option.id === nextSort)) {
      updateParams({ sort: nextSort, cursor: undefined });
    }
  };

  const goToCursor = (cursor: string) => {
    updateParams({ cursor });
  };

  const resetPagination = () => {
    updateParams({ cursor: undefined });
  };

  const metadataCount = devices.filter(
    (device) => device.metadata?.owner || device.metadata?.location || device.metadata?.notes
  ).length;

  const selectedDevice = devices.find((device) => device.id === selectedId);
  const [facts, setFacts] = useState<DeviceFacts | null>(null);
  const [factsStatus, setFactsStatus] = useState<AsyncStatus>('idle');
  const [factsError, setFactsError] = useState<string | null>(null);
  const factsAbortRef = useRef<AbortController | null>(null);

  const [historyEvents, setHistoryEvents] = useState<DeviceChangeEvent[]>([]);
  const [historyCursor, setHistoryCursor] = useState<string | null>(null);
  const [historyStatus, setHistoryStatus] = useState<AsyncStatus>('idle');
  const [historyError, setHistoryError] = useState<string | null>(null);
  const historyAbortRef = useRef<AbortController | null>(null);

  const loadHistoryPage = useCallback(
    async (deviceId: string, cursor?: string | null, reset = false) => {
      if (!deviceId) {
        return;
      }
      historyAbortRef.current?.abort();
      const controller = new AbortController();
      historyAbortRef.current = controller;
      setHistoryStatus('loading');
      setHistoryError(null);

      const params = new URLSearchParams();
      params.set('limit', HISTORY_PAGE_LIMIT.toString());
      if (cursor) {
        params.set('cursor', cursor);
      }

      try {
        const query = params.toString();
        const response = await fetch(
          `/api/v1/devices/${deviceId}/history${query ? `?${query}` : ''}`,
          { cache: 'no-store', signal: controller.signal }
        );
        if (controller.signal.aborted) {
          return;
        }
        if (!response.ok) {
          throw new Error(`History request failed: ${response.status}`);
        }
        const data = (await response.json()) as DeviceChangeFeed;
        if (selectedDevice?.id !== deviceId) {
          return;
        }
        setHistoryEvents((previous) => (reset ? data.events : [...previous, ...data.events]));
        setHistoryCursor(data.cursor ?? null);
        setHistoryStatus('success');
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        if (selectedDevice?.id !== deviceId) {
          return;
        }
        const message = error instanceof Error ? error.message : 'Unable to load device history.';
        setHistoryError(message);
        setHistoryStatus('error');
      } finally {
        if (historyAbortRef.current === controller) {
          historyAbortRef.current = null;
        }
      }
    },
    [selectedDevice?.id]
  );

  useEffect(() => {
    if (!selectedDevice) {
      setHistoryEvents([]);
      setHistoryCursor(null);
      setHistoryStatus('idle');
      setHistoryError(null);
      historyAbortRef.current?.abort();
      return;
    }
    setHistoryEvents([]);
    setHistoryCursor(null);
    setHistoryError(null);
    void loadHistoryPage(selectedDevice.id, undefined, true);
  }, [selectedDevice?.id, loadHistoryPage]);

  useEffect(() => {
    if (!selectedDevice) {
      setFacts(null);
      setFactsStatus('idle');
      setFactsError(null);
      factsAbortRef.current?.abort();
      return;
    }
    const controller = new AbortController();
    factsAbortRef.current?.abort();
    factsAbortRef.current = controller;
    setFacts(null);
    setFactsStatus('loading');
    setFactsError(null);
    const deviceId = selectedDevice.id;

    (async () => {
      try {
        const response = await fetch(`/api/v1/devices/${deviceId}/facts`, {
          cache: 'no-store',
          signal: controller.signal
        });
        if (controller.signal.aborted) {
          return;
        }
        if (!response.ok) {
          throw new Error(`Facts request failed: ${response.status}`);
        }
        const data = (await response.json()) as DeviceFacts;
        if (selectedDevice?.id !== deviceId) {
          return;
        }
        setFacts(data);
        setFactsStatus('success');
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        if (selectedDevice?.id !== deviceId) {
          return;
        }
        const message = error instanceof Error ? error.message : 'Unable to load device facts.';
        setFactsError(message);
        setFactsStatus('error');
        setFacts(null);
      } finally {
        if (factsAbortRef.current === controller) {
          factsAbortRef.current = null;
        }
      }
    })();

    return () => {
      controller.abort();
    };
  }, [selectedDevice?.id]);
  const handleLoadMoreHistory = () => {
    if (!selectedDevice || !historyCursor || historyStatus === 'loading') {
      return;
    }
    void loadHistoryPage(selectedDevice.id, historyCursor);
  };
  const currentStatusLabel = STATUS_OPTIONS.find((option) => option.id === currentStatus)?.label ?? 'All devices';
  const currentSortLabel = SORT_OPTIONS.find((option) => option.id === currentSort)?.label ?? 'Newest created';
  const currentSearchHint = currentSearch ? ` · matching “${currentSearch}”` : '';
  const nextCursor = devicePage.cursor;
  const hasCursorParam = Boolean(searchParams?.get('cursor'));

  const copyText = async (text: string) => {
    const value = text.trim();
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
    } catch {
      // Fallback for older browsers / restricted contexts.
      const textarea = document.createElement('textarea');
      textarea.value = value;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
    }
  };

  const handleReset = () => {
    setSearchInput('');
    router.push(pathname);
  };

  const seenWithinSeconds = Number(searchParams?.get('seen_within_seconds') ?? 3600);
  const changedWithinSeconds = Number(searchParams?.get('changed_within_seconds') ?? 86400);
  const nowMs = Date.now();
  const isOnline = (lastSeenAt?: string | null) => {
    if (!lastSeenAt) return false;
    const ts = Date.parse(lastSeenAt);
    if (!Number.isFinite(ts)) return false;
    return nowMs-ts <= seenWithinSeconds * 1000;
  };
  const isRecentlyChanged = (lastChangeAt?: string | null) => {
    if (!lastChangeAt) return false;
    const ts = Date.parse(lastChangeAt);
    if (!Number.isFinite(ts)) return false;
    return nowMs-ts <= changedWithinSeconds * 1000;
  };

  return (
    <section className="devicesDashboard">
      <div className="devicesDashboardTop">
        <DiscoveryPanel status={discoveryStatus} readOnly={isReadOnly} />
        <ImportExportPanel readOnly={isReadOnly} />
      </div>

      <div className="devicesDashboardControls">
        <form onSubmit={handleSearchSubmit} className="devicesDashboardSearch">
          <Field>
            <Label htmlFor="device-search">Search devices</Label>
            <div style={{ display: 'flex', gap: 8 }}>
              <Input
                id="device-search"
                type="search"
                value={searchInput}
                onChange={(event: ChangeEvent<HTMLInputElement>) => setSearchInput(event.target.value)}
                placeholder="Filter by name, owner, location, or ID"
                className="devicesSearch"
              />
              <Button type="submit">Search</Button>
              <Button type="button" className="btnPill" onClick={handleReset}>
                Reset
              </Button>
            </div>
          </Field>
        </form>

        <div className="devicesDashboardFilterButtons" role="group" aria-label="Device status filter">
          {STATUS_OPTIONS.map((option) => {
            const isActive = currentStatus === option.id;
            const activeClass = isActive ? 'btnPillActive' : undefined;
            return (
              <Button
                key={option.id}
                type="button"
                onClick={() => handleStatusToggle(option.id)}
                className={activeClass ? `btnPill ${activeClass}` : 'btnPill'}
                aria-pressed={isActive}
              >
                {option.label}
              </Button>
            );
          })}
        </div>

        <Field>
          <Label htmlFor="device-sort">Sort</Label>
          <Select id="device-sort" value={currentSort} onChange={handleSortChange} className="devicesSortSelect">
            {SORT_OPTIONS.map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      <div className="hint">
        Showing {devices.length} devices{currentSearchHint} · {currentStatusLabel} · sorted by {currentSortLabel} · {metadataCount}{' '}
        with metadata
      </div>

      <div className="devicesDashboardMain">
        <div className="devicesList">
          {devices.length === 0 ? (
            <EmptyState title="No devices match these filters">
              Try adjusting the search, status, or sort options.
            </EmptyState>
          ) : (
            devices.map((device) => {
              const isSelected = selectedId === device.id;
              const owner = device.metadata?.owner;
              const location = device.metadata?.location;
              const mutedClass = isSelected ? undefined : 'deviceSelectMuted';
              const online = isOnline(device.last_seen_at);
              const changed = isRecentlyChanged(device.last_change_at);

              return (
                <div
                  key={device.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedId(device.id)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      setSelectedId(device.id);
                    }
                  }}
                  aria-pressed={isSelected}
                  className={isSelected ? 'card deviceSelect deviceSelectActive' : 'card deviceSelect'}
                >
                  <div className="split" style={{ alignItems: 'baseline' }}>
                    <span style={{ fontSize: 16, fontWeight: 800 }}>{device.display_name ?? '(unnamed device)'}</span>
                    <span className={mutedClass ? `deviceId ${mutedClass}` : 'deviceId'}>{device.id.slice(0, 8)}</span>
                  </div>

                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 8 }}>
                    {online ? <Badge tone="success">Online</Badge> : <Badge tone="neutral">Offline</Badge>}
                    {changed ? <Badge tone="warning">Changed</Badge> : null}
                    {device.primary_ip ? <Badge tone="info">IP {device.primary_ip}</Badge> : null}
                  </div>

                  {owner || location ? (
                    <div style={{ display: 'grid', gap: 4, fontSize: 14, marginTop: 8 }}>
                      {owner ? <div>Owner: {owner}</div> : null}
                      {location ? <div>Location: {location}</div> : null}
                    </div>
                  ) : (
                    <div className={mutedClass ? mutedClass : undefined} style={{ fontSize: 13, marginTop: 8 }}>
                      No metadata yet.
                    </div>
                  )}

                  {device.metadata?.notes ? (
                    <div className={mutedClass ? mutedClass : undefined} style={{ fontSize: 13, marginTop: 8 }}>
                      Notes: {device.metadata.notes}
                    </div>
                  ) : null}

                  <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' }}>
                    <Button type="button" className="btnPill" onClick={(e: MouseEvent<HTMLButtonElement>) => {
                      e.stopPropagation();
                      router.push(`/devices/${device.id}`);
                    }}>
                      Open
                    </Button>
                    <Button type="button" className="btnPill" onClick={(e: MouseEvent<HTMLButtonElement>) => {
                      e.stopPropagation();
                      void copyText(device.id);
                    }}>
                      Copy ID
                    </Button>
                    <Button
                      type="button"
                      className="btnPill"
                      disabled={!device.primary_ip}
                      onClick={(e: MouseEvent<HTMLButtonElement>) => {
                        e.stopPropagation();
                        if (device.primary_ip) void copyText(device.primary_ip);
                      }}
                    >
                      Copy IP
                    </Button>
                  </div>
                </div>
              );
            })
          )}

          <div className="devicesListFooter" style={{ padding: '8px 0', gap: 8, display: 'flex', alignItems: 'center' }}>
            {hasCursorParam && (
              <Button type="button" onClick={resetPagination} className="btnPill">
                First page
              </Button>
            )}
            <Button type="button" onClick={() => nextCursor && goToCursor(nextCursor)} disabled={!nextCursor}>
              Next page
            </Button>
          </div>
        </div>

        <div className="stack">
          {selectedDevice ? (
            <>
              <Card>
                <CardBody>
                  <div className="stack" style={{ gap: 10 }}>
                    <div className="split" style={{ alignItems: 'center' }}>
                      <div>
                        <p className="kicker">Overview</p>
                        <h3 style={{ margin: '6px 0 0', fontSize: 22 }}>{selectedDevice.display_name ?? '(unnamed device)'}</h3>
                      </div>
                      <Badge tone="info">ID {selectedDevice.id.slice(0, 8)}</Badge>
                    </div>

                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                      {isOnline(selectedDevice.last_seen_at) ? <Badge tone="success">Online</Badge> : <Badge tone="neutral">Offline</Badge>}
                      {isRecentlyChanged(selectedDevice.last_change_at) ? <Badge tone="warning">Changed</Badge> : null}
                      {selectedDevice.primary_ip ? <Badge tone="info">IP {selectedDevice.primary_ip}</Badge> : null}
                    </div>

                    <div style={{ display: 'grid', gap: 4, fontSize: 13, color: 'var(--muted)' }}>
                      <div>Last seen: {formatDateTime(selectedDevice.last_seen_at)}</div>
                      <div>Last changed: {formatDateTime(selectedDevice.last_change_at)}</div>
                      <div>
                        {factsStatus === 'loading'
                          ? 'Refreshing facts…'
                          : factsStatus === 'error'
                          ? factsError ?? 'Unable to refresh discovery facts.'
                          : factsStatus === 'success'
                          ? `Facts refreshed ${formatDateTime(facts?.snmp?.updated_at ?? selectedDevice.last_change_at)}`
                          : 'Facts will sync shortly.'}
                      </div>
                    </div>

                    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 10 }}>
                      <Button
                        type="button"
                        className="btnPill"
                        onClick={(event: MouseEvent<HTMLButtonElement>) => {
                          event.stopPropagation();
                          router.push(`/devices/${selectedDevice.id}`);
                        }}
                      >
                        Open
                      </Button>
                      <Button
                        type="button"
                        className="btnPill"
                        onClick={(event: MouseEvent<HTMLButtonElement>) => {
                          event.stopPropagation();
                          void copyText(selectedDevice.id);
                        }}
                      >
                        Copy ID
                      </Button>
                      <Button
                        type="button"
                        className="btnPill"
                        disabled={!selectedDevice.primary_ip}
                        onClick={(event: MouseEvent<HTMLButtonElement>) => {
                          event.stopPropagation();
                          if (selectedDevice.primary_ip) void copyText(selectedDevice.primary_ip);
                        }}
                      >
                        Copy IP
                      </Button>
                    </div>
                  </div>
                </CardBody>
              </Card>

              <Card>
                <CardBody>
                  <div className="stack" style={{ gap: 10 }}>
                    <div>
                      <p className="kicker">Facts</p>
                      <p className="hint">Details from the discovery/enrichment pipeline.</p>
                    </div>
                    {factsStatus === 'loading' && <div className="hint">Loading discovery facts…</div>}
                    {factsStatus === 'error' && (
                      <div className="alert alertWarning">
                        {factsError ?? 'Unable to load discovery facts for this device.'}
                      </div>
                    )}
                    {factsStatus === 'success' && facts ? (
                      <div className="stack" style={{ gap: 12 }}>
                        <div>
                          <h4 style={{ margin: '0 0 6px' }}>IP addresses</h4>
                          {facts.ips.length === 0 ? (
                            <div className="hint">No IP addresses recorded yet.</div>
                          ) : (
                            <div className="stack" style={{ gap: 6 }}>
                              {facts.ips.map((ip) => (
                                <div key={`${ip.ip}-${ip.updated_at}`} className="split" style={{ alignItems: 'center' }}>
                                  <div>
                                    <strong>{ip.ip}</strong>
                                    <div className="hint" style={{ fontSize: 12 }}>
                                      {ip.interface_name ?? 'Unknown interface'}
                                    </div>
                                  </div>
                                  <div className="hint">{formatDateTime(ip.updated_at)}</div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div>
                          <h4 style={{ margin: '0 0 6px' }}>MAC addresses</h4>
                          {facts.macs.length === 0 ? (
                            <div className="hint">No MAC addresses recorded yet.</div>
                          ) : (
                            <div className="stack" style={{ gap: 6 }}>
                              {facts.macs.map((mac) => (
                                <div key={`${mac.mac}-${mac.updated_at}`} className="split" style={{ alignItems: 'center' }}>
                                  <div>
                                    <strong>{mac.mac}</strong>
                                    <div className="hint" style={{ fontSize: 12 }}>
                                      {mac.interface_name ?? 'Unknown interface'}
                                    </div>
                                  </div>
                                  <div className="hint">{formatDateTime(mac.updated_at)}</div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div>
                          <h4 style={{ margin: '0 0 6px' }}>Interfaces</h4>
                          {facts.interfaces.length === 0 ? (
                            <div className="hint">No interfaces available yet.</div>
                          ) : (
                            <div className="stack" style={{ gap: 6 }}>
                              {facts.interfaces.map((iface) => (
                                <div key={iface.id} style={{ border: '1px solid var(--border)', borderRadius: 10, padding: 10 }}>
                                  <div className="split" style={{ alignItems: 'center' }}>
                                    <span style={{ fontWeight: 700 }}>{iface.name ?? '(unnamed interface)'}</span>
                                    <span className="hint">{formatDateTime(iface.updated_at)}</span>
                                  </div>
                                  <div className="hint" style={{ fontSize: 12, display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                                    IFIndex: {iface.ifindex ?? 'n/a'} · Admin: {iface.admin_status ?? 'n/a'} · Oper: {iface.oper_status ?? 'n/a'} · Speed:{' '}
                                    {iface.speed_bps ? `${(iface.speed_bps / 1_000_000).toFixed(1)} Mb/s` : 'n/a'}
                                  </div>
                                  {iface.descr ? <div className="hint" style={{ fontSize: 12 }}>Description: {iface.descr}</div> : null}
                                  {iface.alias ? <div className="hint" style={{ fontSize: 12 }}>Alias: {iface.alias}</div> : null}
                                  {iface.pvid ? (
                                    <div className="hint" style={{ fontSize: 12 }}>
                                      PVID: {iface.pvid} (observed {formatDateTime(iface.pvid_observed_at)})
                                    </div>
                                  ) : null}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div>
                          <h4 style={{ margin: '0 0 6px' }}>Services</h4>
                          {facts.services.length === 0 ? (
                            <div className="hint">No service scan results yet.</div>
                          ) : (
                            <div className="stack" style={{ gap: 6 }}>
                              {facts.services.map((service, index) => (
                                <div key={`${service.observed_at}-${service.port}-${index}`} className="split" style={{ alignItems: 'center' }}>
                                  <div>
                                    <strong>
                                      {service.protocol?.toUpperCase() ?? 'TCP/UDP'} {service.port ?? 'port?' }
                                    </strong>
                                    <div className="hint" style={{ fontSize: 12 }}>
                                      {service.name ?? 'unnamed'} · {service.state ?? 'state unknown'} · {service.source ?? 'unknown source'}
                                    </div>
                                  </div>
                                  <div className="hint">{formatDateTime(service.observed_at)}</div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        {facts.snmp ? (
                          <div>
                            <h4 style={{ margin: '0 0 6px' }}>SNMP snapshot</h4>
                            <div className="stack" style={{ gap: 4, fontSize: 13 }}>
                              {facts.snmp.sys_name ? <div>sysName: {facts.snmp.sys_name}</div> : null}
                              {facts.snmp.sys_descr ? <div>sysDescr: {facts.snmp.sys_descr}</div> : null}
                              {facts.snmp.sys_location ? <div>sysLocation: {facts.snmp.sys_location}</div> : null}
                              {facts.snmp.sys_contact ? <div>sysContact: {facts.snmp.sys_contact}</div> : null}
                              {facts.snmp.last_success_at ? (
                                <div>Last success: {formatDateTime(facts.snmp.last_success_at)}</div>
                              ) : null}
                              {facts.snmp.last_error ? <div className="hint">Last error: {facts.snmp.last_error}</div> : null}
                            </div>
                          </div>
                        ) : (
                          <div className="hint">SNMP snapshot has not been captured yet.</div>
                        )}

                        {facts.links.length > 0 ? (
                          <div>
                            <h4 style={{ margin: '0 0 6px' }}>Links</h4>
                            <div className="stack" style={{ gap: 6 }}>
                              {facts.links.map((link) => (
                                <div key={link.id} className="split" style={{ alignItems: 'center' }}>
                                  <div>
                                    <strong>{link.link_type ?? 'link'}</strong>
                                    <div className="hint" style={{ fontSize: 12 }}>
                                      Peer: {link.peer_device_id} · Source: {link.source}
                                    </div>
                                  </div>
                                  <div className="hint">{formatDateTime(link.updated_at)}</div>
                                </div>
                              ))}
                            </div>
                          </div>
                        ) : (
                          <div className="hint">No adjacency links observed yet.</div>
                        )}
                      </div>
                    ) : null}
                  </div>
                </CardBody>
              </Card>

              <Card>
                <CardBody>
                  <div className="stack" style={{ gap: 10 }}>
                    <div>
                      <p className="kicker">Metadata</p>
                      <p className="hint">Edit metadata and select friendly display names.</p>
                    </div>
                    <DeviceNameCandidatesPanel
                      deviceId={selectedDevice.id}
                      currentDisplayName={selectedDevice.display_name ?? null}
                      readOnly={isReadOnly}
                    />
                    <DeviceMetadataEditor device={selectedDevice} readOnly={isReadOnly} />
                  </div>
                </CardBody>
              </Card>

              <Card>
                <CardBody>
                  <div className="stack" style={{ gap: 10 }}>
                    <div className="split" style={{ alignItems: 'center' }}>
                      <div>
                        <p className="kicker">History timeline</p>
                        <p className="hint">Powered by the Phase 9 change feed.</p>
                      </div>
                      {historyCursor ? (
                        <Button type="button" onClick={handleLoadMoreHistory} disabled={historyStatus === 'loading'}>
                          Load more
                        </Button>
                      ) : historyEvents.length > 0 ? (
                        <div className="hint" style={{ fontSize: 13 }}>
                          All available events loaded
                        </div>
                      ) : null}
                    </div>
                    {historyStatus === 'loading' && historyEvents.length === 0 && <div className="hint">Loading history…</div>}
                    {historyError ? <div className="alert alertWarning">{historyError}</div> : null}
                    {historyEvents.length === 0 && historyStatus !== 'loading' ? (
                      <EmptyState title="No history yet">
                        Discovery has not recorded any events for this device.
                      </EmptyState>
                    ) : historyEvents.length > 0 ? (
                      <div className="stack" style={{ gap: 10 }}>
                        {historyEvents.map((event) => (
                          <div
                            key={event.event_id}
                            className="card"
                            style={{ padding: 12, borderRadius: 10 }}
                          >
                            <div className="split" style={{ alignItems: 'center' }}>
                              <div>
                                <strong>{event.summary}</strong>
                                <div className="hint" style={{ fontSize: 12 }}>
                                  {event.kind}
                                </div>
                              </div>
                              <div className="hint">{formatDateTime(event.event_at)}</div>
                            </div>
                            {event.details && Object.keys(event.details).length > 0 ? (
                              <div style={{ marginTop: 8, fontSize: 13 }}>
                                {Object.entries(event.details).map(([key, value]) => (
                                  <div key={key}>
                                    <span style={{ fontWeight: 600 }}>{key}:</span>{' '}
                                    {typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value ?? '—')}
                                  </div>
                                ))}
                              </div>
                            ) : null}
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </div>
                </CardBody>
              </Card>
            </>
          ) : (
            <Card>
              <CardBody>
                <EmptyState title="Select a device">Pick a device on the left to view details, facts, and history.</EmptyState>
              </CardBody>
            </Card>
          )}

          <CreateDeviceForm readOnly={isReadOnly} />
        </div>
      </div>
    </section>
  );
}
