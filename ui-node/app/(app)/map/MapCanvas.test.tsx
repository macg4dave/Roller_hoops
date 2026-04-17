import '@testing-library/jest-dom';

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, test } from 'vitest';

import { MapCanvas } from './MapCanvas';
import { MapLayoutProvider } from './MapLayoutContext';
import { MapSelectionProvider } from './MapSelectionContext';

import type { components } from '@/lib/api-types';

type MapProjection = components['schemas']['MapProjection'];

describe('MapCanvas', () => {
  test('renders regions as summary tiles and expands to preview nodes', () => {
    const projection: MapProjection = {
      layer: 'l3',
      focus: null,
      guidance: null,
      regions: [{ id: '10.0.1.0/24', kind: 'subnet', label: '10.0.1.0/24', parent_region_id: null, meta: null }],
      nodes: [
        {
          id: 'device-a',
          kind: 'device',
          label: 'router-1',
          primary_region_id: '10.0.1.0/24',
          region_ids: ['10.0.1.0/24'],
          meta: null
        },
        {
          id: 'device-b',
          kind: 'device',
          label: null,
          primary_region_id: '10.0.1.0/24',
          region_ids: ['10.0.1.0/24'],
          meta: null
        }
      ],
      edges: [],
      inspector: null,
      truncation: {
        regions: { returned: 1, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 2, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    };

    render(
      <MapSelectionProvider>
        <MapLayoutProvider>
          <MapCanvas projection={projection} activeLayerId="l3" currentParams="layer=l3&focusType=device&focusId=device-a" />
        </MapLayoutProvider>
      </MapSelectionProvider>
    );

    expect(screen.getByText('10.0.1.0/24')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'router-1' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Expand' }));

    expect(screen.getByRole('button', { name: 'router-1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'device-b' })).toBeInTheDocument();
  });

  test('renders physical adjacency rows with edge metadata', () => {
    const projection: MapProjection = {
      layer: 'physical',
      focus: { type: 'device', id: 'device-a', label: null },
      guidance: null,
      regions: [],
      nodes: [
        {
          id: 'device-a',
          kind: 'device',
          label: 'Core router',
          primary_region_id: null,
          region_ids: [],
          meta: null
        },
        {
          id: 'device-b',
          kind: 'device',
          label: 'Switch 1',
          primary_region_id: null,
          region_ids: [],
          meta: null
        }
      ],
      edges: [
        {
          id: 'link-1',
          kind: 'link',
          from: 'device-a',
          to: 'device-b',
          label: null,
          meta: { link_type: 'ethernet', source: 'manual', link_key: 'core:sw1' }
        }
      ],
      inspector: null,
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 2, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 1, limit: 80, truncated: false, total: null, warning: null }
      }
    };

    render(
      <MapSelectionProvider>
        <MapLayoutProvider>
          <MapCanvas
            projection={projection}
            activeLayerId="physical"
            currentParams="layer=physical&focusType=device&focusId=device-a"
          />
        </MapLayoutProvider>
      </MapSelectionProvider>
    );

    expect(screen.getByText('Focus device')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Core router' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /switch 1/i })).toBeInTheDocument();
    expect(screen.getByText(/ethernet/i)).toBeInTheDocument();
    expect(screen.getByText(/manual/i)).toBeInTheDocument();
  });

  test('renders primary IP and MAC facts on physical diagram nodes', () => {
    const projection: MapProjection = {
      layer: 'physical',
      focus: { type: 'device', id: 'device-a', label: null },
      guidance: null,
      regions: [],
      nodes: [
        {
          id: 'device-a',
          kind: 'device',
          label: 'Core router',
          primary_region_id: null,
          region_ids: [],
          meta: { device_id: 'device-a', primary_ip: '10.0.1.1', primary_mac: 'aa:bb:cc:dd:ee:01' }
        },
        {
          id: 'device-b',
          kind: 'device',
          label: 'Switch 1',
          primary_region_id: null,
          region_ids: [],
          meta: { device_id: 'device-b', primary_ip: '10.0.1.2', primary_mac: 'aa:bb:cc:dd:ee:02' }
        }
      ],
      edges: [
        {
          id: 'link-1',
          kind: 'link',
          from: 'device-a',
          to: 'device-b',
          label: null,
          meta: { link_type: 'ethernet', source: 'lldp', link_key: 'core:sw1' }
        }
      ],
      inspector: null,
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 2, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 1, limit: 80, truncated: false, total: null, warning: null }
      }
    };

    render(
      <MapSelectionProvider>
        <MapLayoutProvider>
          <MapCanvas
            projection={projection}
            activeLayerId="physical"
            currentParams="layer=physical&focusType=device&focusId=device-a"
          />
        </MapLayoutProvider>
      </MapSelectionProvider>
    );

    expect(screen.getByText('IP 10.0.1.1')).toBeInTheDocument();
    expect(screen.getByText('MAC aa:bb:cc:dd:ee:01')).toBeInTheDocument();

    expect(screen.getByText('IP 10.0.1.2')).toBeInTheDocument();
    expect(screen.getByText('MAC aa:bb:cc:dd:ee:02')).toBeInTheDocument();

    expect(screen.getByText('lldp')).toBeInTheDocument();
  });

  test('renders security zones as regions with device occupants', () => {
    const projection: MapProjection = {
      layer: 'security',
      focus: { type: 'zone', id: 'zone-1', label: 'DMZ' },
      guidance: null,
      regions: [{ id: 'zone-1', kind: 'zone', label: 'DMZ', parent_region_id: null, meta: null }],
      nodes: [
        {
          id: 'device-a',
          kind: 'device',
          label: 'web-1',
          primary_region_id: 'zone-1',
          region_ids: ['zone-1'],
          meta: null
        },
        {
          id: 'device-b',
          kind: 'device',
          label: 'firewall-1',
          primary_region_id: 'zone-1',
          region_ids: ['zone-1'],
          meta: null
        }
      ],
      edges: [],
      inspector: {
        title: 'DMZ',
        identity: [{ label: 'Type', value: 'Zone' }],
        status: [{ label: 'Members', value: '2' }],
        relationships: []
      },
      truncation: {
        regions: { returned: 1, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 2, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    };

    render(
      <MapSelectionProvider>
        <MapLayoutProvider>
          <MapCanvas projection={projection} activeLayerId="security" currentParams="layer=security&focusType=zone&focusId=zone-1" />
        </MapLayoutProvider>
      </MapSelectionProvider>
    );

    expect(screen.getByText('DMZ')).toBeInTheDocument();
    expect(screen.getByText(/members: 2/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'web-1' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Expand' }));

    expect(screen.getByRole('button', { name: 'web-1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'firewall-1' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Drill in' })).toHaveAttribute('href', expect.stringContaining('layer=security'));
  });
});
