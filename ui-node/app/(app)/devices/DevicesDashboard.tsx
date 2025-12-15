'use client';

import { useEffect, useMemo, useState } from 'react';

import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Field, Label } from '@/app/_components/ui/Field';
import { Input } from '@/app/_components/ui/Inputs';

import { CreateDeviceForm } from './CreateDeviceForm';
import { DeviceNameCandidatesPanel } from './DeviceNameCandidatesPanel';
import { DeviceMetadataEditor } from './DeviceMetadataEditor';
import { DiscoveryPanel } from './DiscoveryPanel';
import { ImportExportPanel } from './ImportExportPanel';
import type { Device, DiscoveryStatus } from './types';

type SessionUser = {
  username: string;
  role: string;
};

const FILTER_OPTIONS = [
  { id: 'all', label: 'All devices' },
  { id: 'withMetadata', label: 'With metadata' },
  { id: 'withoutMetadata', label: 'Without metadata' }
] as const;

type FilterMode = (typeof FILTER_OPTIONS)[number]['id'];

type Props = {
  devices: Device[];
  discoveryStatus: DiscoveryStatus;
  currentUser: SessionUser;
};

export function DevicesDashboard({ devices, discoveryStatus, currentUser }: Props) {
  const [search, setSearch] = useState('');
  const [filterMode, setFilterMode] = useState<FilterMode>('all');
  const [selectedId, setSelectedId] = useState<string | undefined>(() => devices[0]?.id);
  const isReadOnly = currentUser.role === 'read-only';

  useEffect(() => {
    if (devices.length === 0) {
      setSelectedId(undefined);
      return;
    }
    setSelectedId((current) => {
      if (current && devices.some((device) => device.id === current)) {
        return current;
      }
      return devices[0].id;
    });
  }, [devices]);

  const filteredDevices = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return devices.filter((device) => {
      const meta = device.metadata;
      const normalized = [
        device.display_name ?? '',
        device.id,
        meta?.owner ?? '',
        meta?.location ?? '',
        meta?.notes ?? ''
      ]
        .map((value) => value.toLowerCase())
        .join(' ');
      const matchesSearch = needle === '' || normalized.includes(needle);
      const hasMetadata = !!(meta?.owner || meta?.location || meta?.notes);
      let matchesFilter = true;
      if (filterMode === 'withMetadata') {
        matchesFilter = hasMetadata;
      } else if (filterMode === 'withoutMetadata') {
        matchesFilter = !hasMetadata;
      }
      return matchesSearch && matchesFilter;
    });
  }, [devices, filterMode, search]);

  useEffect(() => {
    if (filteredDevices.length === 0) {
      setSelectedId(undefined);
      return;
    }
    setSelectedId((current) => {
      if (current && filteredDevices.some((device) => device.id === current)) {
        return current;
      }
      return filteredDevices[0].id;
    });
  }, [filteredDevices]);

  const selectedDevice = filteredDevices.find((device) => device.id === selectedId);

  const metadataCount = devices.filter((device) => device.metadata?.owner || device.metadata?.location || device.metadata?.notes).length;

  return (
    <section className="devicesDashboard">
      <div className="devicesDashboardTop">
        <DiscoveryPanel status={discoveryStatus} readOnly={isReadOnly} />
        <ImportExportPanel readOnly={isReadOnly} />
      </div>

      <div className="devicesDashboardControls">
        <Field>
          <Label htmlFor="device-search">Search devices</Label>
          <Input
            id="device-search"
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Filter by name, owner, location, or ID"
            className="devicesSearch"
          />
        </Field>

        <div className="devicesDashboardFilterButtons" role="group" aria-label="Device metadata filter">
          {FILTER_OPTIONS.map((option) => {
            const isActive = filterMode === option.id;
            const activeClass = isActive ? 'btnPillActive' : undefined;
            return (
              <Button
                key={option.id}
                type="button"
                onClick={() => setFilterMode(option.id)}
                className={activeClass ? `btnPill ${activeClass}` : 'btnPill'}
                aria-pressed={isActive}
              >
                {option.label}
              </Button>
            );
          })}
        </div>
      </div>

      <div className="hint">
        Showing {filteredDevices.length} of {devices.length} devices · {metadataCount} with metadata · {devices.length - metadataCount} without
        metadata
      </div>

      <div className="devicesDashboardMain">
        <div className="devicesList">
          {filteredDevices.length === 0 ? (
            <EmptyState title="No devices match these filters">
              Try relaxing the search text or switching the metadata filter.
            </EmptyState>
          ) : (
            filteredDevices.map((device) => {
              const isSelected = selectedId === device.id;
              const owner = device.metadata?.owner;
              const location = device.metadata?.location;
              const mutedClass = isSelected ? undefined : 'deviceSelectMuted';

              return (
                <button
                  key={device.id}
                  type="button"
                  onClick={() => setSelectedId(device.id)}
                  aria-pressed={isSelected}
                  className={isSelected ? 'card deviceSelect deviceSelectActive' : 'card deviceSelect'}
                >
                  <div className="split" style={{ alignItems: 'baseline' }}>
                    <span style={{ fontSize: 16, fontWeight: 800 }}>{device.display_name ?? '(unnamed device)'}</span>
                    <span className={mutedClass ? `deviceId ${mutedClass}` : 'deviceId'}>{device.id.slice(0, 8)}</span>
                  </div>

                  {owner || location ? (
                    <div style={{ display: 'grid', gap: 4, fontSize: 14 }}>
                      {owner ? <div>Owner: {owner}</div> : null}
                      {location ? <div>Location: {location}</div> : null}
                    </div>
                  ) : (
                    <div className={mutedClass ? mutedClass : undefined} style={{ fontSize: 13 }}>
                      No metadata yet.
                    </div>
                  )}

                  {device.metadata?.notes ? (
                    <div className={mutedClass ? mutedClass : undefined} style={{ fontSize: 13 }}>
                      Notes: {device.metadata.notes}
                    </div>
                  ) : null}
                </button>
              );
            })
          )}
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
