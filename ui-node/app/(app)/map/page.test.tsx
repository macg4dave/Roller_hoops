import '@testing-library/jest-dom';

import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, test, vi } from 'vitest';

import MapPage from './page';

afterEach(() => {
  vi.unstubAllGlobals();
});

const stubProjectionFetch = (projection: unknown) => {
  const fetchMock = vi.fn(async () => ({
    ok: true,
    status: 200,
    json: async () => projection
  }));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
};

describe('MapPage URL contract', () => {
  test('selects layer from the URL query string', async () => {
    const ui = await MapPage({ searchParams: Promise.resolve({ layer: 'l2' }) });
    render(ui);

    expect(screen.getByRole('link', { name: /l2 \(vlans\)/i })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText(/l2 vlan projection/i)).toBeInTheDocument();
  });

  test('shows a friendly warning for unknown layers', async () => {
    const ui = await MapPage({ searchParams: Promise.resolve({ layer: 'banana' }) });
    render(ui);

    expect(screen.getByText('banana')).toBeInTheDocument();
    expect(screen.getByText(/pick a valid layer to continue/i)).toBeInTheDocument();
  });

  test('redirects to the default layer when missing', async () => {
    await expect(MapPage({ searchParams: Promise.resolve({}) })).rejects.toThrow('NEXT_REDIRECT:/map?layer=l3');
  });

  test('renders inspector from the projection response', async () => {
    const focusId = '550e8400-e29b-41d4-a716-446655440000';
    stubProjectionFetch({
      layer: 'l3',
      focus: { type: 'device', id: focusId, label: null },
      guidance: null,
      regions: [],
      nodes: [],
      edges: [],
      inspector: {
        title: 'Router A',
        identity: [
          { label: 'Type', value: 'Device' },
          { label: 'ID', value: focusId }
        ],
        status: [{ label: 'Layer', value: 'l3' }],
        relationships: [
          { label: 'View in L3', layer: 'l3', focus_type: 'device', focus_id: focusId },
          { label: 'View in L2', layer: 'l2', focus_type: 'device', focus_id: focusId }
        ]
      },
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 0, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'device', focusId })
    });
    render(ui);

    expect(screen.getByText('Router A')).toBeInTheDocument();
    expect(screen.getByText('Type')).toBeInTheDocument();
    expect(screen.getByText('Device')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /view in l2/i })).toHaveAttribute('href', expect.stringContaining('layer=l2'));
  });

  test('warns when focusType is unknown', async () => {
    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'banana', focusId: '123' })
    });
    render(ui);

    expect(screen.getByText(/unknown focus type \"banana\"/i)).toBeInTheDocument();
  });

  test('warns when focusType is provided without focusId', async () => {
    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'device' })
    });
    render(ui);

    expect(screen.getByText(/focus type selected, but focus id is missing/i)).toBeInTheDocument();
  });

  test('warns when focusId is provided without focusType', async () => {
    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusId: '550e8400-e29b-41d4-a716-446655440000' })
    });
    render(ui);

    expect(screen.getByText(/focus id provided, but focus type is missing/i)).toBeInTheDocument();
  });

  test('relationship actions keep focus when switching layers', async () => {
    const focusId = '550e8400-e29b-41d4-a716-446655440000';
    stubProjectionFetch({
      layer: 'l2',
      focus: { type: 'device', id: focusId, label: null },
      guidance: null,
      regions: [],
      nodes: [],
      edges: [],
      inspector: {
        title: 'Device',
        identity: [{ label: 'ID', value: focusId }],
        status: [{ label: 'Layer', value: 'l2' }],
        relationships: [{ label: 'View in L3', layer: 'l3', focus_type: 'device', focus_id: focusId }]
      },
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 0, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l2', focusType: 'device', focusId })
    });
    render(ui);

    const viewInL3 = screen.getByRole('link', { name: /view in l3/i });
    expect(viewInL3).toHaveAttribute('href', expect.stringContaining('layer=l3'));
    expect(viewInL3).toHaveAttribute('href', expect.stringContaining('focusType=device'));
    expect(viewInL3).toHaveAttribute('href', expect.stringContaining(`focusId=${focusId}`));
  });
});
