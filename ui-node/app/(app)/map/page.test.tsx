import '@testing-library/jest-dom';

import { fireEvent, render, screen, within } from '@testing-library/react';
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

  test('relationship actions clear focus when the target layer does not support the focus type', async () => {
    const subnetId = '10.0.1.0/24';

    stubProjectionFetch({
      layer: 'l3',
      focus: { type: 'subnet', id: subnetId, label: subnetId },
      guidance: null,
      regions: [],
      nodes: [],
      edges: [],
      inspector: {
        title: subnetId,
        identity: [{ label: 'Type', value: 'Subnet' }],
        status: [{ label: 'Layer', value: 'l3' }],
        relationships: [
          { label: 'View in Physical', layer: 'physical', focus_type: 'subnet', focus_id: subnetId }
        ]
      },
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 0, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'subnet', focusId: subnetId })
    });
    render(ui);

    const viewInPhysical = screen.getByRole('link', { name: /view in physical/i });
    const href = viewInPhysical.getAttribute('href');
    expect(href).not.toBeNull();
    expect(href).toContain('layer=physical');
    expect(href).not.toContain('focusType=');
    expect(href).not.toContain('focusId=');
  });

  test('layer switch links clear focus when the target layer does not support the focus type', async () => {
    stubProjectionFetch({
      layer: 'l2',
      focus: { type: 'vlan', id: '10', label: 'VLAN 10' },
      guidance: null,
      regions: [],
      nodes: [],
      edges: [],
      inspector: {
        title: 'VLAN 10',
        identity: [{ label: 'Type', value: 'VLAN' }],
        status: [{ label: 'Layer', value: 'l2' }],
        relationships: []
      },
      truncation: {
        regions: { returned: 0, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 0, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l2', focusType: 'vlan', focusId: '10' })
    });
    render(ui);

    const l3Link = screen.getByRole('link', { name: /l3 \(subnets\)/i });
    const href = l3Link.getAttribute('href');
    expect(href).not.toBeNull();
    expect(href).toContain('layer=l3');
    expect(href).not.toContain('focusType=');
    expect(href).not.toContain('focusId=');
  });

  test('layer switch links preserve focus when the target layer supports the focus type', async () => {
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
        relationships: []
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

    const l3Link = screen.getByRole('link', { name: /l3 \(subnets\)/i });
    const href = l3Link.getAttribute('href');
    expect(href).not.toBeNull();
    expect(href).toContain('layer=l3');
    expect(href).toContain('focusType=device');
    expect(href).toContain(`focusId=${focusId}`);
  });

  test('clicking a canvas node updates inspector selection without changing focus', async () => {
    const focusId = '550e8400-e29b-41d4-a716-446655440000';
    const peerId = '550e8400-e29b-41d4-a716-446655440111';

    stubProjectionFetch({
      layer: 'l3',
      focus: { type: 'device', id: focusId, label: null },
      guidance: null,
      regions: [{ id: '10.0.1.0/24', kind: 'subnet', label: '10.0.1.0/24', parent_region_id: null, meta: null }],
      nodes: [
        {
          id: focusId,
          kind: 'device',
          label: 'Focus Device',
          primary_region_id: '10.0.1.0/24',
          region_ids: ['10.0.1.0/24'],
          meta: null
        },
        {
          id: peerId,
          kind: 'device',
          label: 'Peer Device',
          primary_region_id: '10.0.1.0/24',
          region_ids: ['10.0.1.0/24'],
          meta: null
        }
      ],
      edges: [],
      inspector: {
        title: 'Focus Device',
        identity: [{ label: 'ID', value: focusId }],
        status: [{ label: 'Layer', value: 'l3' }],
        relationships: [{ label: 'View in L2', layer: 'l2', focus_type: 'device', focus_id: focusId }]
      },
      truncation: {
        regions: { returned: 1, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 2, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'device', focusId })
    });
    render(ui);

    const focusSection = screen.getByText('Focus').closest('.mapInspectorSection');
    expect(focusSection).not.toBeNull();
    expect(within(focusSection as HTMLElement).getByText(new RegExp(focusId))).toBeInTheDocument();
    expect(within(focusSection as HTMLElement).queryByText(new RegExp(peerId))).not.toBeInTheDocument();

    const identitySection = screen.getByText('Identity').closest('.mapInspectorSection');
    expect(identitySection).not.toBeNull();
    expect(within(identitySection as HTMLElement).getByText('Focus Device')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Expand' }));
    fireEvent.click(screen.getByRole('button', { name: 'Peer Device' }));

    expect(within(identitySection as HTMLElement).getByText('Peer Device')).toBeInTheDocument();
    expect(within(focusSection as HTMLElement).getByText(new RegExp(focusId))).toBeInTheDocument();
    expect(within(focusSection as HTMLElement).queryByText(new RegExp(peerId))).not.toBeInTheDocument();

    const relationshipsSection = screen.getByText('Relationships').closest('.mapInspectorSection');
    expect(relationshipsSection).not.toBeNull();

    const drillIn = within(relationshipsSection as HTMLElement).getByRole('link', { name: /drill in/i });
    expect(drillIn).toHaveAttribute('href', expect.stringContaining(`focusId=${peerId}`));
    expect(drillIn).toHaveAttribute('href', expect.stringContaining('layer=l3'));
  });

  test('clicking a canvas region updates inspector selection and keeps drill-in explicit', async () => {
    const focusId = '550e8400-e29b-41d4-a716-446655440000';
    const subnetId = '10.0.1.0/24';
    const encodedSubnetId = encodeURIComponent(subnetId);

    stubProjectionFetch({
      layer: 'l3',
      focus: { type: 'device', id: focusId, label: null },
      guidance: null,
      regions: [{ id: subnetId, kind: 'subnet', label: subnetId, parent_region_id: null, meta: null }],
      nodes: [
        {
          id: focusId,
          kind: 'device',
          label: 'Focus Device',
          primary_region_id: subnetId,
          region_ids: [subnetId],
          meta: null
        }
      ],
      edges: [],
      inspector: {
        title: 'Focus Device',
        identity: [{ label: 'ID', value: focusId }],
        status: [{ label: 'Layer', value: 'l3' }],
        relationships: [{ label: 'View in L2', layer: 'l2', focus_type: 'device', focus_id: focusId }]
      },
      truncation: {
        regions: { returned: 1, limit: 8, truncated: false, total: null, warning: null },
        nodes: { returned: 1, limit: 120, truncated: false, total: null, warning: null },
        edges: { returned: 0, limit: 80, truncated: false, total: null, warning: null }
      }
    });

    const ui = await MapPage({
      searchParams: Promise.resolve({ layer: 'l3', focusType: 'device', focusId })
    });
    render(ui);

    const canvasPanel = screen.getByText('Canvas').closest('.mapCanvasPanel');
    expect(canvasPanel).not.toBeNull();

    const regionTitle = within(canvasPanel as HTMLElement).getByText(subnetId);
    const regionCard = regionTitle.closest('.mapRegionCard');
    expect(regionCard).not.toBeNull();

    fireEvent.click(regionCard as HTMLElement);

    const identitySection = screen.getByText('Identity').closest('.mapInspectorSection');
    expect(identitySection).not.toBeNull();
    expect(within(identitySection as HTMLElement).getByText(subnetId)).toBeInTheDocument();
    expect(within(identitySection as HTMLElement).getByText('Subnet')).toBeInTheDocument();

    const relationshipsSection = screen.getByText('Relationships').closest('.mapInspectorSection');
    expect(relationshipsSection).not.toBeNull();

    const drillIn = within(relationshipsSection as HTMLElement).getByRole('link', { name: /drill in/i });
    expect(drillIn).toHaveAttribute('href', expect.stringContaining('layer=l3'));
    expect(drillIn).toHaveAttribute('href', expect.stringContaining('focusType=subnet'));
    expect(drillIn).toHaveAttribute('href', expect.stringContaining(`focusId=${encodedSubnetId}`));
  });
});
