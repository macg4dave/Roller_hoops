'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';

import { Alert } from '@/app/_components/ui/Alert';
import { Button } from '@/app/_components/ui/Button';
import type { components } from '@/lib/api-types';

import { useMapSelection } from './MapSelectionContext';
import { useOptionalMapProjection } from './MapProjectionContext';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];
type MapProjection = components['schemas']['MapProjection'];
type MapRegion = components['schemas']['MapRegion'];
type MapNode = components['schemas']['MapNode'];
type MapEdge = components['schemas']['MapEdge'];

const SUMMARY_SAMPLE_LIMIT = 3;
const OCCUPANT_EXPANDED_LIMIT = 25;

function resolveFocusTypeFromRegion(region: MapRegion): MapFocusType | undefined {
  switch (region.kind) {
    case 'subnet':
      return 'subnet';
    case 'vlan':
      return 'vlan';
    case 'zone':
      return 'zone';
    case 'device':
      return 'device';
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

function resolveNodeHoverTitle(node: MapNode): string {
  const title = resolveNodeTitle(node);
  if (node.label?.trim() && title !== node.id) {
    return `${title} (${node.id})`;
  }
  return title;
}

function resolveRegionTitle(region: MapRegion): string {
  const label = region.label?.trim();
  if (label) {
    return label;
  }
  return region.id;
}

function resolveRegionHoverTitle(region: MapRegion): string {
  const title = resolveRegionTitle(region);
  if (region.label?.trim() && title !== region.id) {
    return `${title} (${region.id})`;
  }
  return title;
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

function resolveEdgeMetaString(edge: MapEdge, key: string): string | undefined {
  const meta = edge.meta;
  if (!meta || typeof meta !== 'object') {
    return undefined;
  }
  const value = (meta as Record<string, unknown>)[key];
  if (typeof value !== 'string') {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

export function MapCanvas({
  projection: projectionProp,
  activeLayerId,
  currentParams
}: {
  projection: MapProjection;
  activeLayerId: MapLayer;
  currentParams: string;
}) {
  const { selection, setSelection, clearSelection } = useMapSelection();
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const projectionContext = useOptionalMapProjection();
  const projection = projectionContext?.projection ?? projectionProp;

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

  const nodesByMembership = useMemo(() => {
    const membership = new Map<string, MapNode[]>();
    for (const node of nodes) {
      for (const regionId of node.region_ids ?? []) {
        const list = membership.get(regionId) ?? [];
        list.push(node);
        membership.set(regionId, list);
      }
    }
    return membership;
  }, [nodes]);

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

  const isProjectionTruncated = Boolean(
    projection.truncation?.regions?.truncated ||
      projection.truncation?.nodes?.truncated ||
      projection.truncation?.edges?.truncated
  );

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

  const isPhysical = activeLayerId === 'physical';
  const showRegions = !isPhysical && regions.length > 0;

  const physicalLinks = useMemo(() => {
    if (!isPhysical) {
      return [];
    }

    const focusId = projection.focus?.id;

    const links = (projection.edges ?? []).map((edge) => {
      const from = edge.from;
      const to = edge.to;
      const peerId = focusId ? (from === focusId ? to : to === focusId ? from : to) : to;
      const peerNode = nodeById.get(peerId);
      const linkType = resolveEdgeMetaString(edge, 'link_type');
      const source = resolveEdgeMetaString(edge, 'source');
      const linkKey = resolveEdgeMetaString(edge, 'link_key');

      return {
        edge,
        peerId,
        peerLabel: peerNode ? resolveNodeTitle(peerNode) : peerId,
        linkType,
        source,
        linkKey
      };
    });

    return links.sort((a, b) => {
      const labelA = a.peerLabel.toLowerCase();
      const labelB = b.peerLabel.toLowerCase();
      if (labelA !== labelB) {
        return labelA < labelB ? -1 : 1;
      }
      if (a.peerId !== b.peerId) {
        return a.peerId < b.peerId ? -1 : 1;
      }
      return a.edge.id < b.edge.id ? -1 : a.edge.id > b.edge.id ? 1 : 0;
    });
  }, [isPhysical, projection.edges, projection.focus?.id, nodeById]);

  const focusId = projection.focus?.id;

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
          {isProjectionTruncated ? (
            <Alert tone="info">Projection capped for readability. Drill in to a region to see a smaller, complete slice.</Alert>
          ) : null}
        </div>
      ) : null}

      {isPhysical ? (
        <div className="mapPhysicalView">
          <div className="mapPhysicalSection">
            <div className="mapPhysicalHeading">Focus device</div>
            {focusId ? (
              <button
                type="button"
                className={`mapPhysicalFocus${selection?.kind === 'node' && selection.id === focusId ? ' mapPhysicalFocusSelected' : ''}`}
                onClick={() => setSelection({ kind: 'node', id: focusId })}
                title={nodeById.get(focusId) ? resolveNodeHoverTitle(nodeById.get(focusId)!) : focusId}
              >
                {nodeById.get(focusId) ? resolveNodeTitle(nodeById.get(focusId)!) : focusId}
              </button>
            ) : (
              <p className="mapRegionEmpty">No focus returned.</p>
            )}
          </div>

          <div className="mapPhysicalSection">
            <div className="mapPhysicalHeading">Links</div>
            {physicalLinks.length === 0 ? (
              <p className="mapRegionEmpty">No links returned for this focus.</p>
            ) : (
              <ul className="mapPhysicalLinkList">
                {physicalLinks.map((link) => {
                  const selected = selection?.kind === 'node' && selection.id === link.peerId;
                  const meta: string[] = [];
                  if (link.linkType) meta.push(link.linkType);
                  if (link.source) meta.push(link.source);
                  if (link.linkKey) meta.push(link.linkKey);

                  return (
                    <li key={link.edge.id} className="mapPhysicalLinkListItem">
                      <button
                        type="button"
                        className={`mapPhysicalLinkRow${selected ? ' mapPhysicalLinkRowSelected' : ''}`}
                        onClick={() => setSelection({ kind: 'node', id: link.peerId })}
                        title={nodeById.get(link.peerId) ? resolveNodeHoverTitle(nodeById.get(link.peerId)!) : link.peerLabel}
                      >
                        <span className="mapPhysicalLinkLabel">{link.peerLabel}</span>
                        {meta.length > 0 ? <span className="mapPhysicalLinkMeta">{meta.join(' • ')}</span> : null}
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>
      ) : showRegions ? (
        <div className="mapRegionStack" role="list">
          {regions.map((region) => {
            const placementNodes = nodesByPlacement.placements.get(region.id) ?? [];
            const isSelected = selection?.kind === 'region' && selection.id === region.id;
            const focusType = resolveFocusTypeFromRegion(region);
            const memberNodes = nodesByMembership.get(region.id) ?? [];
            const memberCount = memberNodes.length;
            const placementCount = placementNodes.length;
            const unplacedCount = Math.max(0, memberCount - placementCount);
            const canPreview = placementCount > 0;
            const expandedKey = canPreview ? (expanded[region.id] ?? false) : false;
            const visible = expandedKey ? placementNodes.slice(0, OCCUPANT_EXPANDED_LIMIT) : [];
            const hiddenCount = Math.max(0, placementNodes.length - visible.length);
            const sampleTitles = memberNodes.slice(0, SUMMARY_SAMPLE_LIMIT).map(resolveNodeTitle);
            const sampleRemaining = Math.max(0, memberCount - sampleTitles.length);

            return (
              <div
                key={region.id}
                className={`mapRegionCard${isSelected ? ' mapRegionCardSelected' : ''}`}
                role="listitem"
                tabIndex={0}
                title={resolveRegionHoverTitle(region)}
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
                    <div className="mapRegionMeta">
                      Members: {memberCount} • Placed: {placementCount}
                      {unplacedCount > 0 ? ` • +${unplacedCount} not placed here` : ''}
                    </div>
                  </div>
                  <div className="mapRegionActions">
                    <Button
                      type="button"
                      disabled={!canPreview}
                      onClick={(event) => {
                        event.stopPropagation();
                        if (!canPreview) {
                          return;
                        }
                        setExpanded((prev) => ({ ...prev, [region.id]: !expandedKey }));
                      }}
                    >
                      {expandedKey ? 'Collapse' : 'Expand'}
                    </Button>
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

                {!expandedKey ? (
                  <div className="mapRegionSummary">
                    {memberCount === 0 ? (
                      <p className="mapRegionSummaryText">No members returned for this region.</p>
                    ) : (
                      <p className="mapRegionSummaryText">
                        {sampleTitles.join(', ')}
                        {sampleRemaining > 0 ? ` (+${sampleRemaining} more)` : ''}
                      </p>
                    )}
                    <p className="mapRegionSummaryHint">
                      {canPreview
                        ? 'Expand to preview placed nodes. Drill in to see full membership (and avoid duplicate nodes in the overview).'
                        : 'No nodes are placed here in the overview (to avoid duplicates). Drill in to see full membership.'}
                    </p>
                  </div>
                ) : visible.length > 0 ? (
                  <div className="mapNodeGrid">
                    {visible.map((node) => {
                      const nodeSelected = selection?.kind === 'node' && selection.id === node.id;
                      return (
                        <button
                          key={node.id}
                          type="button"
                          className={`mapNodeChip${nodeSelected ? ' mapNodeChipSelected' : ''}`}
                          title={resolveNodeHoverTitle(node)}
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

                {expandedKey && hiddenCount > 0 ? (
                  <p className="mapRegionTruncation">
                    Showing {visible.length} of {placementNodes.length} placed nodes. Drill in to view full membership.
                  </p>
                ) : null}
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
                    title={resolveNodeHoverTitle(node)}
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
            <Button type="button" onClick={clearSelection}>
              Clear
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
