import '@testing-library/jest-dom';

import { render, screen } from '@testing-library/react';
import { describe, expect, test } from 'vitest';

import { MapCanvas } from './MapCanvas';

import type { components } from '@/lib/api-types';

type MapProjection = components['schemas']['MapProjection'];

describe('MapCanvas', () => {
  test('renders regions and node chips', () => {
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

    render(<MapCanvas projection={projection} activeLayerId="l3" currentParams="layer=l3&focusType=device&focusId=device-a" />);

    expect(screen.getByText('10.0.1.0/24')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'router-1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'device-b' })).toBeInTheDocument();
  });
});

