import '@testing-library/jest-dom';

import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, test, vi } from 'vitest';

import { DiscoveryRunLogViewer } from './DiscoveryRunLogViewer';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('DiscoveryRunLogViewer', () => {
  test('renders logs and loads additional entries', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        logs: [
          {
            id: 2,
            level: 'info',
            message: 'Worker pinged gateway',
            created_at: '2025-12-15T01:00:00.000Z'
          }
        ],
        cursor: null
      })
    });

    vi.stubGlobal('fetch', fetchMock);

    render(
      <DiscoveryRunLogViewer
        runId="run-1"
        initialLogs={[
          {
            id: 1,
            level: 'info',
            message: 'Run started',
            created_at: '2025-12-15T00:55:00.000Z'
          }
        ]}
        initialCursor="cursor-1"
      />
    );

    expect(screen.getByText('Run started')).toBeInTheDocument();

    const loadMore = screen.getByRole('button', { name: /load more logs/i });
    fireEvent.click(loadMore);

    expect(await screen.findByText('Worker pinged gateway')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/discovery/runs/run-1/logs?limit=40&cursor=cursor-1', {
      cache: 'no-store'
    });
  });

  test('shows an error when fetching logs fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 503, json: async () => ({}) }));

    render(
      <DiscoveryRunLogViewer
        runId="run-2"
        initialLogs={[]}
        initialCursor="cursor-2"
      />
    );

    const loadMore = screen.getByRole('button', { name: /load more logs/i });
    fireEvent.click(loadMore);

    expect(await screen.findByText('Failed to load run logs (503)')).toBeInTheDocument();
  });
});
