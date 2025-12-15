import '@testing-library/jest-dom';
import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';

vi.mock('./CreateDeviceForm', () => ({
  CreateDeviceForm: () => null
}));
vi.mock('./DeviceNameCandidatesPanel', () => ({
  DeviceNameCandidatesPanel: () => null
}));
vi.mock('./DeviceMetadataEditor', () => ({
  DeviceMetadataEditor: () => null
}));
vi.mock('./DiscoveryPanel', () => ({
  DiscoveryPanel: () => null
}));
vi.mock('./ImportExportPanel', () => ({
  ImportExportPanel: () => null
}));

import { DevicesDashboard } from './DevicesDashboard';
import type { DeviceFacts, DeviceChangeFeed } from './types';

type DashboardProps = Parameters<typeof DevicesDashboard>[0];

const testDevice = {
  id: 'device-1',
  display_name: 'Edge Switch',
  primary_ip: '10.0.0.5',
  last_seen_at: '2025-12-14T12:00:00.000Z',
  last_change_at: '2025-12-13T12:00:00.000Z',
  metadata: {
    owner: 'Networking',
    location: 'Data Center',
    notes: 'Manually curated'
  }
};

const baseProps: DashboardProps = {
  devicePage: { devices: [testDevice], cursor: null },
  discoveryStatus: { status: 'idle' },
  currentUser: { username: 'tester', role: 'admin' },
  initialFilters: { search: '', status: 'all', sort: 'created_desc' }
};

const factsResponse: DeviceFacts = {
  device_id: testDevice.id,
  ips: [],
  macs: [],
  interfaces: [],
  services: [],
  snmp: null,
  links: []
};

const historyPageOne: DeviceChangeFeed = {
  events: [
    {
      event_id: 'evt-1',
      device_id: testDevice.id,
      event_at: '2025-12-13T10:00:00.000Z',
      kind: 'ip_address',
      summary: 'Added IP 10.0.0.5',
      details: { ip: '10.0.0.5' }
    }
  ],
  cursor: 'cursor-1'
};

const historyPageTwo: DeviceChangeFeed = {
  events: [
    {
      event_id: 'evt-2',
      device_id: testDevice.id,
      event_at: '2025-12-12T08:45:00.000Z',
      kind: 'ip_address',
      summary: 'Removed IP 10.0.0.4',
      details: { ip: '10.0.0.4' }
    }
  ],
  cursor: null
};

const renderDashboard = (overrides?: Partial<DashboardProps>) => {
  return render(<DevicesDashboard {...baseProps} {...overrides} />);
};

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const stubFetchWithHistory = (pages: DeviceChangeFeed[]) => {
  let historyCallCount = 0;
  const fetchMock = vi.fn(async (input) => {
    const url = typeof input === 'string' ? input : input.url;
    if (url.includes('/facts')) {
      return {
        ok: true,
        status: 200,
        json: async () => factsResponse
      };
    }
    if (url.includes('/history')) {
      const response = pages[Math.min(historyCallCount, pages.length - 1)];
      historyCallCount += 1;
      return {
        ok: true,
        status: 200,
        json: async () => response
      };
    }
    return {
      ok: true,
      status: 200,
      json: async () => ({})
    };
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
};

test('renders the history timeline and loads additional events', async () => {
  const fetchMock = stubFetchWithHistory([historyPageOne, historyPageTwo]);
  renderDashboard();

  const firstEvent = await screen.findByText('Added IP 10.0.0.5');
  expect(firstEvent).toBeInTheDocument();

  const loadMoreButton = await screen.findByRole('button', { name: /Load more/i });
  fireEvent.click(loadMoreButton);

  const secondEvent = await screen.findByText('Removed IP 10.0.0.4');
  expect(secondEvent).toBeInTheDocument();

  const loadedIndicator = await screen.findByText(/All available events loaded/i);
  expect(loadedIndicator).toBeInTheDocument();

  expect(fetchMock).toHaveBeenCalled();
});

test('shows an error when the history feed fails to load', async () => {
  const fetchMock = vi.fn(async (input) => {
    const url = typeof input === 'string' ? input : input.url;
    if (url.includes('/facts')) {
      return {
        ok: true,
        status: 200,
        json: async () => factsResponse
      };
    }
    if (url.includes('/history')) {
      return {
        ok: false,
        status: 503,
        json: async () => ({})
      };
    }
    return {
      ok: true,
      status: 200,
      json: async () => ({})
    };
  });
  vi.stubGlobal('fetch', fetchMock);

  renderDashboard();

  const errorMessage = await screen.findByText('History request failed: 503');
  expect(errorMessage).toBeInTheDocument();
});
