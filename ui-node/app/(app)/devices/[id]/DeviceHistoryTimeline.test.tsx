import '@testing-library/jest-dom';

import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';

import { DeviceHistoryTimeline } from './DeviceHistoryTimeline';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

test('renders an empty state when there are no history events', () => {
  render(<DeviceHistoryTimeline deviceId="dev-1" initialEvents={[]} initialCursor={null} />);

  expect(screen.getByText('No history yet')).toBeInTheDocument();
});

test('loads additional history pages using the Phase 9 history endpoint', async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString();
    if (!url.includes('/api/v1/devices/dev-1/history')) {
      throw new Error(`Unexpected fetch url: ${url}`);
    }

    return {
      ok: true,
      status: 200,
      json: async () => ({
        events: [
          {
            event_id: 'evt-2',
            device_id: 'dev-1',
            event_at: '2025-12-12T08:45:00.000Z',
            kind: 'ip_address',
            summary: 'Removed IP 10.0.0.4',
            details: { ip: '10.0.0.4' }
          }
        ],
        cursor: null
      })
    } as unknown as Response;
  });

  vi.stubGlobal('fetch', fetchMock);

  render(
    <DeviceHistoryTimeline
      deviceId="dev-1"
      initialEvents={[
        {
          event_id: 'evt-1',
          device_id: 'dev-1',
          event_at: '2025-12-13T10:00:00.000Z',
          kind: 'ip_address',
          summary: 'Added IP 10.0.0.5',
          details: { ip: '10.0.0.5' }
        }
      ]}
      initialCursor="cursor-1"
      limit={25}
    />
  );

  expect(await screen.findByText('Added IP 10.0.0.5')).toBeInTheDocument();

  const loadMore = await screen.findByRole('button', { name: /load more/i });
  fireEvent.click(loadMore);

  expect(await screen.findByText('Removed IP 10.0.0.4')).toBeInTheDocument();
  expect(await screen.findByText(/All available events loaded/i)).toBeInTheDocument();

  expect(fetchMock).toHaveBeenCalledTimes(1);
});

test('shows an error message when loading more history fails', async () => {
  const fetchMock = vi.fn(async () => {
    return {
      ok: false,
      status: 503,
      json: async () => ({})
    } as unknown as Response;
  });

  vi.stubGlobal('fetch', fetchMock);

  render(
    <DeviceHistoryTimeline
      deviceId="dev-1"
      initialEvents={[
        {
          event_id: 'evt-1',
          device_id: 'dev-1',
          event_at: '2025-12-13T10:00:00.000Z',
          kind: 'metadata',
          summary: 'Metadata updated',
          details: { owner: 'Networking' }
        }
      ]}
      initialCursor="cursor-1"
    />
  );

  const loadMore = await screen.findByRole('button', { name: /load more/i });
  fireEvent.click(loadMore);

  expect(await screen.findByText('History request failed: 503')).toBeInTheDocument();
});
