'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';

import { Alert } from '@/app/_components/ui/Alert';
import { Button } from '@/app/_components/ui/Button';
import type { components } from '@/lib/api-types';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];
type MapProjection = components['schemas']['MapProjection'];
type MapRegion = components['schemas']['MapRegion'];
type MapNode = components['schemas']['MapNode'];

type Selection =
  | { kind: 'region'; id: string }
  | { kind: 'node'; id: string };

const OCCUPANT_PREVIEW_LIMIT = 8;
const OCCUPANT_EXPANDED_LIMIT = 25;

function resolveFocusTypeFromRegion(region: MapRegion): MapFocusType | undefined {
  switch (region.kind) {
    case 'subnet':
      return 'subnet';
    case 'vlan':
      return 'vlan';
    case 'zone':
      return 'zone';
    default:
      return undefined;
  }
}

function resolveFocusTypeFromNode(node: MapNode): MapFocusType | undefined {
  switch (node.kind) {
    case 'device':
      return 'device';
    case 'service':
      return 'service';
    default:
      return undefined;
  }
}

function resolveNodeTitle(node: MapNode): string {
  const label = node.label?.trim();
  if (label) {
    return label;
  }
  return node.id;
}

function resolveRegionTitle(region: MapRegion): string {
  const label = region.label?.trim();
  if (label) {
    return label;
  }
  return region.id;
}

function resolveNodePlacement(node: MapNode): string | undefined {
  if (node.primary_region_id) {
    return node.primary_region_id;
  }
  const first = node.region_ids[0];
  if (first) {
    return first;
  }
  return undefined;
}

export function MapCanvas({
  projection,
  activeLayerId,
  currentParams
}: {
  projection: MapProjection;
  activeLayerId: MapLayer;
  currentParams: string;
}) {
  const [selection, setSelection] = useState<Selection | null>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const paramsBase = useMemo(() => new URLSearchParams(currentParams), [currentParams]);

  const regions = projection.regions ?? [];
  const nodes = projection.nodes ?? [];

  const regionById = useMemo(() => {
    const map = new Map<string, MapRegion>();
    for (const region of regions) {
      map.set(region.id, region);
    }
    return map;
  }, [regions]);

  const nodeById = useMemo(() => {
    const map = new Map<string, MapNode>();
    for (const node of nodes) {
      map.set(node.id, node);
    }
    return map;
  }, [nodes]);

  const nodesByPlacement = useMemo(() => {
    const placements = new Map<string, MapNode[]>();
    const unplaced: MapNode[] = [];

    for (const node of nodes) {
      const placement = resolveNodePlacement(node);
      if (!placement || !regionById.has(placement)) {
        unplaced.push(node);
        continue;
      }
      const list = placements.get(placement) ?? [];
      list.push(node);
      placements.set(placement, list);
    }

    return { placements, unplaced };
  }, [nodes, regionById]);

  const truncationWarnings = useMemo(() => {
    const warnings: string[] = [];
    const trunc = projection.truncation;
    if (trunc?.regions?.warning) {
      warnings.push(trunc.regions.warning);
    }
    if (trunc?.nodes?.warning) {
      warnings.push(trunc.nodes.warning);
    }
    if (trunc?.edges?.warning) {
      warnings.push(trunc.edges.warning);
    }
    return warnings;
  }, [projection.truncation]);

  function buildHref(next: { layer: MapLayer; focusType?: MapFocusType; focusId?: string }): string {
    const params = new URLSearchParams(paramsBase);
    params.set('layer', next.layer);
    if (next.focusType && next.focusId) {
      params.set('focusType', next.focusType);
      params.set('focusId', next.focusId);
    } else {
      params.delete('focusType');
      params.delete('focusId');
    }
    return `/map?${params.toString()}`;
  }

  const selectionDetails = useMemo(() => {
    if (!selection) {
      return null;
    }
    if (selection.kind === 'region') {
      const region = regionById.get(selection.id);
      if (!region) {
        return null;
      }
      const focusType = resolveFocusTypeFromRegion(region);
      return {
        title: resolveRegionTitle(region),
        focusType,
        focusId: region.id
      };
    }

    const node = nodeById.get(selection.id);
    if (!node) {
      return null;
    }
    const focusType = resolveFocusTypeFromNode(node);
    return {
      title: resolveNodeTitle(node),
      focusType,
      focusId: node.id
    };
  }, [selection, nodeById, regionById]);

  const showRegions = regions.length > 0;

  return (
    <div className="mapCanvasInner">
      {projection.guidance ? <Alert tone="info">{projection.guidance}</Alert> : null}
      {truncationWarnings.length > 0 ? (
        <div className="mapCanvasWarnings">
          {truncationWarnings.map((warning) => (
            <Alert key={warning} tone="warning">
              {warning}
            </Alert>
          ))}
        </div>
      ) : null}

      {showRegions ? (
        <div className="mapRegionStack" role="list">
          {regions.map((region) => {
            const placementNodes = nodesByPlacement.placements.get(region.id) ?? [];
            const isSelected = selection?.kind === 'region' && selection.id === region.id;
            const focusType = resolveFocusTypeFromRegion(region);
            const expandedKey = expanded[region.id] ?? placementNodes.length <= OCCUPANT_PREVIEW_LIMIT;
            const limit = expandedKey ? OCCUPANT_EXPANDED_LIMIT : OCCUPANT_PREVIEW_LIMIT;
            const visible = placementNodes.slice(0, limit);
            const hiddenCount = Math.max(0, placementNodes.length - visible.length);

            return (
              <div
                key={region.id}
                className={`mapRegionCard${isSelected ? ' mapRegionCardSelected' : ''}`}
                role="listitem"
                tabIndex={0}
                onClick={() => setSelection({ kind: 'region', id: region.id })}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    setSelection({ kind: 'region', id: region.id });
                  }
                }}
              >
                <div className="mapRegionHeader">
                  <div>
                    <div className="mapRegionTitle">{resolveRegionTitle(region)}</div>
                    <div className="mapRegionMeta">{placementNodes.length} nodes placed</div>
                  </div>
                  <div className="mapRegionActions">
                    {placementNodes.length > OCCUPANT_PREVIEW_LIMIT ? (
                      <Button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          setExpanded((prev) => ({ ...prev, [region.id]: !expandedKey }));
                        }}
                      >
                        {expandedKey ? 'Collapse' : 'Expand'}
                      </Button>
                    ) : null}
                    <Link
                      className={`btn${focusType ? ' btnPrimary' : ''}`}
                      href={focusType ? buildHref({ layer: activeLayerId, focusType, focusId: region.id }) : '#'}
                      aria-disabled={!focusType}
                      onClick={(event) => {
                        if (!focusType) {
                          event.preventDefault();
                          event.stopPropagation();
                        }
                      }}
                    >
                      Drill in
                    </Link>
                  </div>
                </div>

                {visible.length > 0 ? (
                  <div className="mapNodeGrid">
                    {visible.map((node) => {
                      const nodeSelected = selection?.kind === 'node' && selection.id === node.id;
                      return (
                        <button
                          key={node.id}
                          type="button"
                          className={`mapNodeChip${nodeSelected ? ' mapNodeChipSelected' : ''}`}
                          onClick={(event) => {
                            event.stopPropagation();
                            setSelection({ kind: 'node', id: node.id });
                          }}
                        >
                          <span className="mapNodeChipLabel">{resolveNodeTitle(node)}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <p className="mapRegionEmpty">No placed nodes in this region.</p>
                )}

                {hiddenCount > 0 ? <p className="mapRegionTruncation">Showing {visible.length} of {placementNodes.length} nodes.</p> : null}
              </div>
            );
          })}
        </div>
      ) : (
        <div className="mapNodeCloud" role="list">
          {nodes.length === 0 ? <p className="mapRegionEmpty">No nodes returned for this focus.</p> : null}
          {nodes.length > 0 ? (
            <div className="mapNodeGrid">
              {nodes.slice(0, OCCUPANT_EXPANDED_LIMIT).map((node) => {
                const nodeSelected = selection?.kind === 'node' && selection.id === node.id;
                return (
                  <button
                    key={node.id}
                    type="button"
                    className={`mapNodeChip${nodeSelected ? ' mapNodeChipSelected' : ''}`}
                    onClick={() => setSelection({ kind: 'node', id: node.id })}
                  >
                    <span className="mapNodeChipLabel">{resolveNodeTitle(node)}</span>
                  </button>
                );
              })}
            </div>
          ) : null}
          {nodes.length > OCCUPANT_EXPANDED_LIMIT ? (
            <p className="mapRegionTruncation">Showing {OCCUPANT_EXPANDED_LIMIT} of {nodes.length} nodes.</p>
          ) : null}
        </div>
      )}

      {selectionDetails ? (
        <div className="mapCanvasSelection" role="status">
          <div className="mapCanvasSelectionTitle">{selectionDetails.title}</div>
          <div className="mapCanvasSelectionActions">
            {selectionDetails.focusType ? (
              <Link className="btn btnPrimary" href={buildHref({ layer: activeLayerId, focusType: selectionDetails.focusType, focusId: selectionDetails.focusId })}>
                Open
              </Link>
            ) : (
              <Button type="button" disabled>
                Open
              </Button>
            )}
            <Button type="button" onClick={() => setSelection(null)}>
              Clear
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

