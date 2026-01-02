import '@testing-library/jest-dom';

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { DiscoveryRunList } from './DiscoveryRunList';
import { api } from '@/lib/api-client';

vi.mock('@/lib/api-client', () => ({
  api: {
    GET: vi.fn()
  }
}));

const mockGet = api.GET as ReturnType<typeof vi.fn>;

const createClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
        staleTime: 0
      }
    }
  });

const renderWithClient = (ui: React.ReactElement) => {
  const client = createClient();
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
};

afterEach(() => {
  mockGet.mockReset();
});

describe('DiscoveryRunList', () => {
  test('renders runs and loads additional pages', async () => {
    const firstPage = {
      data: {
        runs: [
          {
            id: 'run-1',
            status: 'succeeded',
            scope: '10.0.0.0/24',
            stats: { stage: 'completed', preset: 'fast', tags: ['ports', 'snmp'] },
            started_at: '2025-12-15T00:00:00.000Z',
            completed_at: '2025-12-15T00:01:00.000Z',
            last_error: null
          }
        ],
        cursor: 'cursor-1'
      },
      error: undefined
    };

    const nextPage = {
      data: {
        runs: [
          {
            id: 'run-2',
            status: 'failed',
            scope: '10.0.1.0/24',
            stats: { stage: 'mismatched', preset: 'deep' },
            started_at: '2025-12-14T12:00:00.000Z',
            completed_at: null,
            last_error: 'Timeout'
          }
        ],
        cursor: null
      },
      error: undefined
    };

    mockGet.mockResolvedValueOnce(firstPage).mockResolvedValueOnce(nextPage);

    renderWithClient(<DiscoveryRunList limit={1} />);

    expect(await screen.findByText(/run-1/)).toBeInTheDocument();
    expect(screen.getByText('Scope: 10.0.0.0/24 • Preset: Fast • Tags: Ports (nmap), SNMP')).toBeInTheDocument();

    const loadMore = screen.getByRole('button', { name: /load more runs/i });
    fireEvent.click(loadMore);

    await waitFor(() => expect(mockGet).toHaveBeenCalledTimes(2));
    expect(await screen.findByText(/run-2/)).toBeInTheDocument();
    expect(screen.getByText('Timeout')).toBeInTheDocument();
  });

  test('shows an error when the run list request fails', async () => {
    mockGet.mockResolvedValue({ data: { runs: [] }, error: { message: 'Bad request' } });

    renderWithClient(<DiscoveryRunList limit={2} />);

    expect(await screen.findByText(/Bad request/)).toBeInTheDocument();
  });
});
