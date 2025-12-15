'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';

import { CreateDeviceForm } from './CreateDeviceForm';
import { DeviceNameCandidatesPanel } from './DeviceNameCandidatesPanel';
import { DeviceMetadataEditor } from './DeviceMetadataEditor';
import { DiscoveryPanel } from './DiscoveryPanel';
import { ImportExportPanel } from './ImportExportPanel';
import { LogoutButton } from './LogoutButton';
import type { Device, DiscoveryStatus } from './types';
import type { SessionUser } from '../../lib/auth/session';

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
    <section style={{ display: 'grid', gap: 16, marginTop: 16 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12
        }}
      >
          <div style={{ display: 'grid', gap: 4 }}>
            <div style={{ fontSize: 14, color: '#111827' }}>Signed in as {currentUser.username}</div>
            <div style={{ fontSize: 12, color: '#4b5563' }}>Role: {currentUser.role}</div>
          </div>
        <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
          <Link href="/auth/account" style={{ color: '#111827', fontWeight: 600 }}>
            Account
          </Link>
          <LogoutButton />
        </div>
      </div>
      {isReadOnly ? (
        <div style={{ color: '#92400e', fontSize: 13, marginTop: 4 }}>
          Read-only role: mutation controls have been disabled. Contact your admin to change this role.
        </div>
      ) : null}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
          gap: 16
        }}
      >
          <DiscoveryPanel status={discoveryStatus} readOnly={isReadOnly} />
          <ImportExportPanel readOnly={isReadOnly} />
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr auto',
          alignItems: 'center',
          gap: 12,
          flexWrap: 'wrap'
        }}
      >
        <div style={{ display: 'grid', gap: 8 }}>
          <label style={{ fontSize: 12, fontWeight: 600, color: '#374151' }} htmlFor="device-search">
            Search devices
          </label>
          <input
            id="device-search"
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Filter by name, owner, location, or ID"
            style={{
              padding: '10px 12px',
              borderRadius: 8,
              border: '1px solid #d1d5db',
              width: 320
            }}
          />
        </div>
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
          {FILTER_OPTIONS.map((option) => {
            const isActive = filterMode === option.id;
            return (
              <button
                key={option.id}
                type="button"
                onClick={() => setFilterMode(option.id)}
                style={{
                  borderRadius: 999,
                  padding: '6px 14px',
                  border: '1px solid #d1d5db',
                  background: isActive ? '#111827' : '#fff',
                  color: isActive ? '#fff' : '#374151',
                  cursor: 'pointer',
                  fontWeight: 600,
                  fontSize: 13
                }}
              >
                {option.label}
              </button>
            );
          })}
        </div>
      </div>

      <div
        style={{
          fontSize: 14,
          color: '#555'
        }}
      >
        Showing {filteredDevices.length} of {devices.length} devices · {metadataCount} with metadata · {devices.length - metadataCount}{' '}
        without metadata
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(0, 1fr) minmax(320px, 360px)',
          gap: 16
        }}
      >
        <div
          style={{
            display: 'grid',
            gap: 12
          }}
        >
          {filteredDevices.length === 0 ? (
            <p style={{ color: '#6b7280' }}>No devices match those filters — try relaxing the search.</p>
          ) : (
            filteredDevices.map((device) => {
              const isSelected = selectedId === device.id;
              const owner = device.metadata?.owner;
              const location = device.metadata?.location;
              return (
                <button
                  key={device.id}
                  type="button"
                  onClick={() => setSelectedId(device.id)}
                  aria-pressed={isSelected}
                  style={{
                    border: '1px solid #e0e0e0',
                    borderRadius: 10,
                    padding: '16px 18px',
                    textAlign: 'left',
                    background: isSelected ? '#111827' : '#fff',
                    color: isSelected ? '#f8fafc' : '#111827',
                    cursor: 'pointer',
                    display: 'grid',
                    gap: 6
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 6 }}>
                    <span style={{ fontSize: 16, fontWeight: 700 }}>{device.display_name ?? '(unnamed device)'}</span>
                    <span
                      style={{
                        fontSize: 12,
                        letterSpacing: '0.08em',
                        textTransform: 'uppercase',
                        color: isSelected ? '#cbd5f5' : '#64748b'
                      }}
                    >
                      {device.id.slice(0, 8)}
                    </span>
                  </div>
                  {owner || location ? (
                    <div style={{ display: 'grid', gap: 4, fontSize: 14 }}>
                      {owner ? <div>Owner: {owner}</div> : null}
                      {location ? <div>Location: {location}</div> : null}
                    </div>
                  ) : (
                    <div style={{ color: isSelected ? '#e2e8f0' : '#6b7280', fontSize: 13 }}>No metadata yet.</div>
                  )}
                  {device.metadata?.notes ? (
                    <div style={{ fontSize: 13, color: isSelected ? '#d1d5ff' : '#475569' }}>
                      Notes: {device.metadata.notes}
                    </div>
                  ) : null}
                </button>
              );
            })
          )}
        </div>

        <div style={{ display: 'grid', gap: 16 }}>
          <div
            style={{
              border: '1px solid #e0e0e0',
              borderRadius: 10,
              padding: 16,
              minHeight: 320
            }}
          >
            {selectedDevice ? (
              <div style={{ display: 'grid', gap: 10 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                  <div>
                    <p style={{ margin: 0, fontSize: 12, letterSpacing: '0.08em', textTransform: 'uppercase', color: '#4b5563' }}>
                      Device detail
                    </p>
                    <h3 style={{ margin: '6px 0 0', fontSize: 22 }}>{selectedDevice.display_name ?? '(unnamed device)'}</h3>
                  </div>
                  <span
                    style={{
                      fontSize: 12,
                      color: '#475569',
                      background: '#f8fafc',
                      borderRadius: 999,
                      padding: '4px 10px'
                    }}
                  >
                    ID {selectedDevice.id.slice(0, 8)}
                  </span>
                </div>

                <div style={{ fontSize: 14, color: '#475569', display: 'grid', gap: 6 }}>
                  {selectedDevice.metadata?.owner ? <div>Owner: {selectedDevice.metadata.owner}</div> : null}
                  {selectedDevice.metadata?.location ? <div>Location: {selectedDevice.metadata.location}</div> : null}
                  {selectedDevice.metadata?.notes ? <div>Notes: {selectedDevice.metadata.notes}</div> : null}
                  {!selectedDevice.metadata && <div>No metadata recorded yet.</div>}
                </div>

                <DeviceNameCandidatesPanel deviceId={selectedDevice.id} currentDisplayName={selectedDevice.display_name ?? null} readOnly={isReadOnly} />
                <DeviceMetadataEditor device={selectedDevice} readOnly={isReadOnly} />
              </div>
            ) : (
              <div style={{ color: '#6b7280' }}>Select a device to see its details and edit metadata.</div>
            )}
          </div>

          <CreateDeviceForm readOnly={isReadOnly} />
        </div>
      </div>
    </section>
  );
}
