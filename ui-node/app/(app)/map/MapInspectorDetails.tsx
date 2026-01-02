'use client';

import Link from 'next/link';
import { useMemo } from 'react';

import { Button } from '@/app/_components/ui/Button';
import type { components } from '@/lib/api-types';

import { useMapSelection } from './MapSelectionContext';
import { useOptionalMapProjection } from './MapProjectionContext';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];
type MapProjection = components['schemas']['MapProjection'];
type MapRegion = components['schemas']['MapRegion'];
type MapNode = components['schemas']['MapNode'];
type MapInspector = components['schemas']['MapInspector'];
type MapInspectorField = components['schemas']['MapInspectorField'];
type MapInspectorRelationship = components['schemas']['MapInspectorRelationship'];

const RELATIONSHIP_LAYER_ACTIONS = [
  { layer: 'physical', label: 'View in Physical' },
  { layer: 'l2', label: 'View in L2' },
  { layer: 'l3', label: 'View in L3' },
  { layer: 'services', label: 'View in Services' },
  { layer: 'security', label: 'View in Security' }
] as const satisfies ReadonlyArray<{ layer: MapLayer; label: string }>;

const LAYER_FOCUS_SUPPORT = {
  physical: ['device'],
  l2: ['device', 'vlan'],
  l3: ['device', 'subnet'],
  services: ['device', 'service'],
  security: ['device', 'zone']
} as const satisfies Record<MapLayer, readonly MapFocusType[]>;

function formatLayerLabel(layer: MapLayer): string {
  switch (layer) {
    case 'l2':
      return 'L2';
    case 'l3':
      return 'L3';
    case 'physical':
      return 'Physical';
    case 'services':
      return 'Services';
    case 'security':
      return 'Security';
    default:
      return layer;
  }
}

function resolveCrossLayerFocus(
  layer: MapLayer,
  focus: { type: MapFocusType; id: string } | undefined
): { type: MapFocusType; id: string } | undefined {
  if (!focus) {
    return undefined;
  }
  const supported: readonly MapFocusType[] = LAYER_FOCUS_SUPPORT[layer] ?? [];
  return supported.includes(focus.type) ? focus : undefined;
}

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

function formatLabel(label?: string | null): string | undefined {
  const trimmed = label?.trim();
  return trimmed ? trimmed : undefined;
}

function formatRegionTitle(region: MapRegion): string {
  return formatLabel(region.label) ?? region.id;
}

function formatNodeTitle(node: MapNode): string {
  return formatLabel(node.label) ?? node.id;
}

function titleCase(value: string): string {
  if (!value) return value;
  return value.slice(0, 1).toUpperCase() + value.slice(1);
}

function formatList(values: string[], limit: number): string {
  if (values.length <= limit) {
    return values.join(', ');
  }
  return `${values.slice(0, limit).join(', ')}, …`;
}

function buildSelectionInspector({
  selection,
  projection,
  activeLayerId,
}: {
  selection: { kind: 'region'; id: string } | { kind: 'node'; id: string };
  projection: MapProjection;
  activeLayerId: MapLayer;
}): MapInspector | null {
  const regions = projection.regions ?? [];
  const nodes = projection.nodes ?? [];

  const regionById = new Map<string, MapRegion>();
  for (const region of regions) {
    regionById.set(region.id, region);
  }

  const nodeById = new Map<string, MapNode>();
  for (const node of nodes) {
    nodeById.set(node.id, node);
  }

  const relationships: MapInspectorRelationship[] = [];
  const identity: MapInspectorField[] = [];
  const status: MapInspectorField[] = [{ label: 'Layer', value: activeLayerId }];

  if (selection.kind === 'region') {
    const region = regionById.get(selection.id);
    if (!region) {
      return null;
    }

    const focusType = resolveFocusTypeFromRegion(region);
    const title = formatRegionTitle(region);
    const kind = titleCase(region.kind);

    identity.push({ label: 'Type', value: kind });
    identity.push({ label: 'ID', value: region.id });
    if (region.label !== region.id) {
      identity.push({ label: 'Label', value: region.label });
    }
    if (region.parent_region_id) {
      identity.push({ label: 'Parent', value: region.parent_region_id });
    }

    const memberCount = nodes.filter((node) => node.region_ids.includes(region.id)).length;
    status.push({ label: 'Members', value: String(memberCount) });

    if (focusType) {
      relationships.push({ label: 'Drill in', layer: activeLayerId, focus_type: focusType, focus_id: region.id });
      for (const action of RELATIONSHIP_LAYER_ACTIONS) {
        relationships.push({ label: action.label, layer: action.layer, focus_type: focusType, focus_id: region.id });
      }
    }

    return { title, identity, status, relationships };
  }

  const node = nodeById.get(selection.id);
  if (!node) {
    return null;
  }

  const focusType = resolveFocusTypeFromNode(node);
  const title = formatNodeTitle(node);
  const kind = titleCase(node.kind);

  identity.push({ label: 'Type', value: kind });
  identity.push({ label: 'ID', value: node.id });
  if (formatLabel(node.label) && node.label !== node.id) {
    identity.push({ label: 'Label', value: formatLabel(node.label) ?? node.id });
  }

  const regionsForNode = node.region_ids
    .map((id) => regionById.get(id))
    .filter((region): region is MapRegion => Boolean(region));

  if (node.primary_region_id) {
    const primaryRegion = regionById.get(node.primary_region_id);
    status.push({ label: 'Primary region', value: primaryRegion ? formatRegionTitle(primaryRegion) : node.primary_region_id });
  }

  if (node.region_ids.length > 0) {
    const regionTitles = regionsForNode.map((region) => formatRegionTitle(region));
    const fallback = node.region_ids.filter((id) => !regionById.has(id));
    const all = [...regionTitles, ...fallback].filter(Boolean);
    status.push({ label: 'Regions', value: String(node.region_ids.length) });
    identity.push({ label: 'Also in', value: formatList(all, 4) });
  } else {
    status.push({ label: 'Regions', value: '0' });
  }

  if (focusType) {
    relationships.push({ label: 'Drill in', layer: activeLayerId, focus_type: focusType, focus_id: node.id });
    for (const action of RELATIONSHIP_LAYER_ACTIONS) {
      relationships.push({ label: action.label, layer: action.layer, focus_type: focusType, focus_id: node.id });
    }
  }

  return { title, identity, status, relationships };
}

export function MapInspectorDetails({
  focus,
  focusTypeLabel,
  projection,
  activeLayerId,
  currentParams
}: {
  focus?: { type: MapFocusType; id: string };
  focusTypeLabel?: string;
  projection?: MapProjection;
  activeLayerId: MapLayer;
  currentParams: string;
}) {
  const { selection } = useMapSelection();
  const projectionContext = useOptionalMapProjection();
  const resolvedProjection = projectionContext?.projection ?? projection;

  const selectionInspector = useMemo(() => {
    if (!selection) {
      return null;
    }
    if (!resolvedProjection) {
      return null;
    }
    return buildSelectionInspector({ selection, projection: resolvedProjection, activeLayerId });
  }, [selection, resolvedProjection, activeLayerId, currentParams]);

  const inspector = selectionInspector ?? resolvedProjection?.inspector ?? null;

  const inspectorTitle = inspector?.title;
  const inspectorIdentity: MapInspectorField[] = inspector?.identity ?? [];
  const inspectorStatus: MapInspectorField[] = inspector?.status ?? [];
  const inspectorRelationships: MapInspectorRelationship[] = inspector?.relationships ?? [];

  const paramsBase = useMemo(() => new URLSearchParams(currentParams), [currentParams]);

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

  return (
    <>
      <div className="mapInspectorSection">
        <div className="mapInspectorHeading">Identity</div>
        <p className="mapInspectorValue">
          {inspectorTitle ?? (focus ? `${focusTypeLabel ?? focus.type} ${focus.id}` : 'No object selected')}
        </p>
        {((selectionInspector && inspectorIdentity.length > 0) || (focus && inspectorIdentity.length > 0)) ? (
          <div className="mapInspectorFieldList">
            {inspectorIdentity.map((field) => (
              <div className="mapInspectorFieldRow" key={`${field.label}:${field.value}`}>
                <span className="mapInspectorFieldLabel">{field.label}</span>
                <span className="mapInspectorFieldValue">{field.value}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="mapInspectorHint">
            When focused, we will show stable identifiers, ownership, and metadata for the focused object.
          </p>
        )}
      </div>

      <div className="mapInspectorSection">
        <div className="mapInspectorHeading">Status</div>
        <p className="mapInspectorValue">
          {focus ? (inspectorStatus.length > 0 ? 'Focus loaded' : 'Focus set') : 'Awaiting focus'}
        </p>
        {focus && inspectorStatus.length > 0 ? (
          <div className="mapInspectorFieldList">
            {inspectorStatus.map((field) => (
              <div className="mapInspectorFieldRow" key={`${field.label}:${field.value}`}>
                <span className="mapInspectorFieldLabel">{field.label}</span>
                <span className="mapInspectorFieldValue">{field.value}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="mapInspectorHint">
            Active focus will show health, last discovery time, and notes once projections are wired.
          </p>
        )}
      </div>

      <div className="mapInspectorSection">
        <div className="mapInspectorHeading">Relationships</div>
        <div className="mapInspectorActions">
          {!focus || inspectorRelationships.length === 0 ? (
            RELATIONSHIP_LAYER_ACTIONS.map(({ layer, label }) => (
              <Button key={layer} variant="default" disabled>
                {label}
              </Button>
            ))
          ) : (
            <>
              {inspectorRelationships.map((rel) => {
                const sourceFocus = { type: rel.focus_type, id: rel.focus_id };
                const targetFocus =
                  rel.layer === activeLayerId ? sourceFocus : resolveCrossLayerFocus(rel.layer, sourceFocus);
                const clearsFocus = rel.layer !== activeLayerId && !targetFocus;

                const href = buildHref({
                  layer: rel.layer,
                  focusType: targetFocus?.type,
                  focusId: targetFocus?.id
                });
                const isActive =
                  rel.layer === activeLayerId &&
                  focus &&
                  targetFocus &&
                  targetFocus.type === focus.type &&
                  targetFocus.id === focus.id;

                return (
                  <Link
                    key={`${rel.layer}:${rel.focus_type}:${rel.focus_id}:${rel.label}`}
                    href={href}
                    className={isActive ? 'btn btnPrimary' : 'btn'}
                    title={
                      clearsFocus
                        ? `The ${formatLayerLabel(rel.layer)} layer does not support ${sourceFocus.type} focus; switching will clear focus.`
                        : undefined
                    }
                  >
                    {rel.label}
                  </Link>
                );
              })}
            </>
          )}
        </div>
        <p className="mapInspectorHint">
          Relationship actions switch layers; focus is preserved when the target layer supports the same focus type.
        </p>
      </div>
    </>
  );
}
