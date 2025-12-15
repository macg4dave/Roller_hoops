import '@testing-library/jest-dom';

import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, test, vi } from 'vitest';

import { AppHeader } from './AppHeader';
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

describe('AppHeader discovery indicator', () => {
  test('shows a discovery indicator when queued/running', async () => {
    mockGet.mockResolvedValue({
      data: { status: 'running', latest_run: { id: 'run-1', status: 'running', started_at: new Date().toISOString() } },
      error: undefined
    });

    renderWithClient(<AppHeader user={{ username: 'alice', role: 'admin' }} />);

    await waitFor(() => expect(mockGet).toHaveBeenCalled());

    expect(await screen.findByRole('link', { name: /discovery running/i })).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
  });

  test('does not show an indicator when idle', async () => {
    mockGet.mockResolvedValue({ data: { status: 'succeeded', latest_run: null }, error: undefined });

    renderWithClient(<AppHeader user={{ username: 'alice', role: 'admin' }} />);

    await waitFor(() => expect(mockGet).toHaveBeenCalled());

    expect(screen.queryByText('running')).not.toBeInTheDocument();
    expect(screen.queryByText('queued')).not.toBeInTheDocument();
  });
});
