'use client';

import { ChangeEvent, FormEvent, MouseEvent, useEffect, useState } from 'react';
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
import type { DevicePage, DiscoveryStatus } from './types';

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
                      setSelectedId(device.id);
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
          <Card>
            <CardBody>
              {selectedDevice ? (
                <div className="stack" style={{ gap: 10 }}>
                  <div className="split" style={{ alignItems: 'center' }}>
                    <div>
                      <p className="kicker">Device detail</p>
                      <h3 style={{ margin: '6px 0 0', fontSize: 22 }}>{selectedDevice.display_name ?? '(unnamed device)'}</h3>
                    </div>
                    <Badge tone="info">ID {selectedDevice.id.slice(0, 8)}</Badge>
                  </div>

                  <div style={{ fontSize: 14, color: 'var(--muted)', display: 'grid', gap: 6 }}>
                    {selectedDevice.metadata?.owner ? <div>Owner: {selectedDevice.metadata.owner}</div> : null}
                    {selectedDevice.metadata?.location ? <div>Location: {selectedDevice.metadata.location}</div> : null}
                    {selectedDevice.metadata?.notes ? <div>Notes: {selectedDevice.metadata.notes}</div> : null}
                    {!selectedDevice.metadata && <div>No metadata recorded yet.</div>}
                  </div>

                  <DeviceNameCandidatesPanel
                    deviceId={selectedDevice.id}
                    currentDisplayName={selectedDevice.display_name ?? null}
                    readOnly={isReadOnly}
                  />
                  <DeviceMetadataEditor device={selectedDevice} readOnly={isReadOnly} />
                </div>
              ) : (
                <EmptyState title="Select a device">Pick a device on the left to view details and edit metadata.</EmptyState>
              )}
            </CardBody>
          </Card>

          <CreateDeviceForm readOnly={isReadOnly} />
        </div>
      </div>
    </section>
  );
}
