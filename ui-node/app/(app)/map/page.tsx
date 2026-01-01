import Link from 'next/link';
import { redirect } from 'next/navigation';

import { Alert } from '@/app/_components/ui/Alert';
import { Button } from '@/app/_components/ui/Button';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Field, Hint, Label } from '@/app/_components/ui/Field';
import { Input, Select } from '@/app/_components/ui/Inputs';
import type { components } from '@/lib/api-types';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];

const LAYER_OPTIONS = [
  { id: 'physical', label: 'Physical', description: 'Cables, racks, and adjacency' },
  { id: 'l2', label: 'L2 (VLANs)', description: 'VLAN grouping and tagged ports' },
  { id: 'l3', label: 'L3 (Subnets)', description: 'Subnets and device membership' },
  { id: 'services', label: 'Services', description: 'Discovered ports and protocol services' },
  { id: 'security', label: 'Security', description: 'Zones, policies, and focus-driven flows' }
] as const satisfies ReadonlyArray<{ id: MapLayer; label: string; description: string }>;

const RELATIONSHIP_LAYER_ACTIONS = [
  { layer: 'physical', label: 'View in Physical' },
  { layer: 'l2', label: 'View in L2' },
  { layer: 'l3', label: 'View in L3' },
  { layer: 'services', label: 'View in Services' },
  { layer: 'security', label: 'View in Security' }
] as const satisfies ReadonlyArray<{ layer: MapLayer; label: string }>;

type LayerId = (typeof LAYER_OPTIONS)[number]['id'];

const FOCUS_TYPE_OPTIONS = [
  { id: 'device', label: 'Device' },
  { id: 'subnet', label: 'Subnet' },
  { id: 'vlan', label: 'VLAN' },
  { id: 'zone', label: 'Zone' },
  { id: 'service', label: 'Service' }
] as const satisfies ReadonlyArray<{ id: MapFocusType; label: string }>;

type FocusType = (typeof FOCUS_TYPE_OPTIONS)[number]['id'];

const LAYER_RENDER_CONFIG: Record<
  LayerId,
  {
    canvasHint: string;
    emptyTitle: string;
  }
> = {
  physical: {
    canvasHint: 'Physical adjacency projection (mocked)',
    emptyTitle: 'Pick a focus to render physical adjacency'
  },
  l2: {
    canvasHint: 'L2 VLAN projection (mocked)',
    emptyTitle: 'Pick a focus to render VLAN membership'
  },
  l3: {
    canvasHint: 'L3 subnet projection (mocked)',
    emptyTitle: 'Pick a focus to render subnet membership'
  },
  services: {
    canvasHint: 'Services projection (mocked)',
    emptyTitle: 'Pick a focus to render discovered services'
  },
  security: {
    canvasHint: 'Security projection (mocked)',
    emptyTitle: 'Pick a focus to render security zones'
  }
};

type RawSearchParams = {
  [key: string]: string | string[] | undefined;
};

function toSingleValue(value?: string | string[]): string | undefined {
  if (!value) {
    return undefined;
  }
  return Array.isArray(value) ? value[0] : value;
}

function toUrlSearchParams(raw: RawSearchParams): URLSearchParams {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(raw)) {
    if (!value) {
      continue;
    }
    if (Array.isArray(value)) {
      for (const entry of value) {
        params.append(key, entry);
      }
      continue;
    }
    params.set(key, value);
  }
  return params;
}

const DEFAULT_LAYER: LayerId = 'l3';

function resolveLayer(raw: string | undefined): { layer?: LayerId; unknown?: string } {
  if (!raw) {
    return { layer: undefined };
  }

  const normalized = raw.trim().toLowerCase();
  if (!normalized) {
    return { layer: undefined };
  }

  const match = LAYER_OPTIONS.find((layer) => layer.id === normalized);
  if (!match) {
    return { layer: undefined, unknown: normalized };
  }

  return { layer: match.id };
}

function resolveFocusType(raw: string | undefined): { focusType?: FocusType; unknown?: string } {
  if (!raw) {
    return { focusType: undefined };
  }

  const normalized = raw.trim().toLowerCase();
  if (!normalized) {
    return { focusType: undefined };
  }

  const match = FOCUS_TYPE_OPTIONS.find((focusType) => focusType.id === normalized);
  if (!match) {
    return { focusType: undefined, unknown: normalized };
  }

  return { focusType: match.id };
}

export default async function MapPage({ searchParams }: { searchParams?: Promise<RawSearchParams> }) {
  const resolvedSearchParams = searchParams ? await searchParams : {};
  const currentParams = toUrlSearchParams(resolvedSearchParams);
  const rawLayer = toSingleValue(resolvedSearchParams.layer);

  if (!rawLayer?.trim()) {
    const nextParams = new URLSearchParams(currentParams);
    nextParams.set('layer', DEFAULT_LAYER);
    redirect(`/map?${nextParams.toString()}`);
  }

  const { layer: resolvedLayerId, unknown: unknownLayer } = resolveLayer(rawLayer);
  const activeLayerId = unknownLayer ? undefined : (resolvedLayerId ?? DEFAULT_LAYER);
  const activeLayer = activeLayerId ? LAYER_OPTIONS.find((layer) => layer.id === activeLayerId) : undefined;
  const activeLayerConfig = activeLayerId ? LAYER_RENDER_CONFIG[activeLayerId] : undefined;

  const rawFocusType = toSingleValue(resolvedSearchParams.focusType);
  const rawFocusId = toSingleValue(resolvedSearchParams.focusId);
  const { focusType: resolvedFocusType, unknown: unknownFocusType } = resolveFocusType(rawFocusType);
  const resolvedFocusId = rawFocusId?.trim() ? rawFocusId.trim() : undefined;
  const hasFocusParams = Boolean(rawFocusType?.trim() || rawFocusId?.trim());

  const focus =
    resolvedFocusType && resolvedFocusId
      ? { type: resolvedFocusType, id: resolvedFocusId }
      : undefined;

  const focusTypeLabel = focus
    ? FOCUS_TYPE_OPTIONS.find((option) => option.id === focus.type)?.label ?? focus.type
    : undefined;

  const focusWarning =
    unknownFocusType
      ? `Unknown focus type "${unknownFocusType}".`
      : resolvedFocusType && !resolvedFocusId
        ? 'Focus type selected, but focus id is missing.'
        : !resolvedFocusType && resolvedFocusId
          ? 'Focus id provided, but focus type is missing.'
          : undefined;

  const clearFocusParams = new URLSearchParams(currentParams);
  clearFocusParams.delete('focusType');
  clearFocusParams.delete('focusId');

  return (
    <div className="mapPage">
      <header className="stack">
        <p className="kicker">Layered explorer</p>
        <h1 className="pageTitle">Network map</h1>
        <p className="pageSubTitle">Select a layer and focus to render your topology without clutter.</p>
      </header>

      <section className="mapShell">
        <aside className="mapPanel mapLayerPanel">
          <div className="mapPanelHeader">
            <div>
              <p className="mapPanelKicker">Layers</p>
              <h2 className="mapPanelTitle">One layer at a time</h2>
            </div>
            <p className="mapPanelHint">Switch layers to reframe the canvas; the inspector stays visible.</p>
          </div>

          <div className="mapLayerList" role="list">
            {unknownLayer ? (
              <Alert tone="warning">
                Unknown layer <strong>{unknownLayer}</strong>. Pick a valid layer to continue.
              </Alert>
            ) : null}
            {LAYER_OPTIONS.map((layer) => {
              const active = layer.id === activeLayerId;
              const nextParams = new URLSearchParams(currentParams);
              nextParams.set('layer', layer.id);
              return (
                <Link
                  key={layer.id}
                  href={`/map?${nextParams.toString()}`}
                  className={`mapLayerItem${active ? ' mapLayerItemActive' : ''}`}
                  aria-current={active ? 'page' : undefined}
                >
                  <span className="mapLayerItemLabel">{layer.label}</span>
                  <span className="mapLayerItemMeta">{layer.description}</span>
                </Link>
              );
            })}
          </div>
        </aside>

        <section
          className="mapPanel mapCanvasPanel"
          key={
            activeLayerId
              ? `${activeLayerId}:${focus?.type ?? 'none'}:${focus?.id ?? 'none'}`
              : unknownLayer
                ? `unknown:${unknownLayer}`
                : 'no-layer'
          }
        >
          <div className="mapCanvasHeader">
            <p className="mapCanvasIntro">Canvas</p>
            <p className="mapCanvasHint">
              {unknownLayer
                ? `Unknown layer "${unknownLayer}".`
                : activeLayerConfig
                  ? activeLayerConfig.canvasHint
                  : 'Select a layer to get started.'}{' '}
              {focus
                ? `Focus: ${focusTypeLabel} ${focus.id}.`
                : focusWarning
                  ? 'No valid focus selected yet.'
                  : 'No focus selected yet.'}
            </p>
          </div>
          <div className="mapCanvasBody">
            <EmptyState
              title={
                unknownLayer
                  ? 'Unknown layer'
                  : activeLayer
                    ? focus
                      ? `Focused on ${focusTypeLabel}`
                      : activeLayerConfig?.emptyTitle ?? `Pick a focus to render the ${activeLayer.label} projection`
                    : 'Select a layer and focus to get started'
              }
            >
              {focus ? (
                <>
                  <p>
                    This milestone locks the deep-link contract. Rendered projections arrive in Phase 14+ (API
                    projections) and Phase 15+ (canvas rendering).
                  </p>
                  <p>
                    Share this URL to reopen the same layer + focus. Use the inspector to adjust focus without drawing
                    the whole network.
                  </p>
                </>
              ) : (
                <>
                  <p>
                    The canvas stays empty until you pick something to focus on. This keeps the view intentional and
                    avoids the spaghetti effect from the mocks.
                  </p>
                  <p>
                    Use the inspector on the right to jump between layers and follow relationships without losing
                    context.
                  </p>
                </>
              )}
            </EmptyState>
          </div>
        </section>

        <section className="mapPanel mapInspectorPanel">
          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Focus</div>
            <p className="mapInspectorValue">
              {focus ? `${focusTypeLabel} ${focus.id}` : hasFocusParams ? 'Incomplete focus' : 'No focus selected'}
            </p>
            <p className="mapInspectorHint">
              Paste an identifier to create a reload-safe deep link. Selecting a layer never draws the whole network.
            </p>

            {focusWarning ? <Alert tone="warning">{focusWarning}</Alert> : null}

            <form action="/map" method="get" className="mapFocusForm">
              <input type="hidden" name="layer" value={activeLayerId ?? DEFAULT_LAYER} />

              <Field>
                <Label htmlFor="focusType">Focus type</Label>
                <Select id="focusType" name="focusType" defaultValue={resolvedFocusType ?? ''}>
                  <option value="">Select…</option>
                  {FOCUS_TYPE_OPTIONS.map((option) => (
                    <option key={option.id} value={option.id}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </Field>

              <Field>
                <Label htmlFor="focusId">Focus id</Label>
                <Input
                  id="focusId"
                  name="focusId"
                  placeholder="UUID / subnet / vlan / zone / service id"
                  defaultValue={resolvedFocusId ?? ''}
                />
                <Hint>Device focus ids are UUIDs. Subnet focus ids are CIDR strings (e.g. 10.0.1.0/24).</Hint>
              </Field>

              <div className="mapInspectorActions">
                <Button type="submit" variant="primary">
                  Apply focus
                </Button>
                {hasFocusParams ? (
                  <Link href={`/map?${clearFocusParams.toString()}`} className="btn">
                    Clear
                  </Link>
                ) : null}
              </div>
            </form>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Identity</div>
            <p className="mapInspectorValue">{focus ? `${focusTypeLabel} ${focus.id}` : 'No object selected'}</p>
            <p className="mapInspectorHint">
              When focused, we will show stable identifiers, ownership, and metadata for the focused object.
            </p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Status</div>
            <p className="mapInspectorValue">{focus ? 'Focus set (mocked projection)' : 'Awaiting focus'}</p>
            <p className="mapInspectorHint">
              Active focus will show health, last discovery time, and notes once projections are wired.
            </p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Relationships</div>
            <div className="mapInspectorActions">
              {RELATIONSHIP_LAYER_ACTIONS.map(({ layer, label }) => {
                if (!focus) {
                  return (
                    <Button key={layer} variant="default" disabled>
                      {label}
                    </Button>
                  );
                }

                const targetParams = new URLSearchParams(currentParams);
                targetParams.set('layer', layer);
                targetParams.set('focusType', focus.type);
                targetParams.set('focusId', focus.id);
                const href = `/map?${targetParams.toString()}`;
                const active = layer === activeLayerId;

                return (
                  <Link key={layer} href={href} className={active ? 'btn btnPrimary' : 'btn'}>
                    {label}
                  </Link>
                );
              })}
            </div>
            <p className="mapInspectorHint">
              Relationship actions will switch layers while keeping the focused object in view.
            </p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Guidance</div>
            <p className="mapInspectorHint">
              URL-driven state: `/map?layer=l3&focusType=device&focusId=...`. No focus is valid and renders an empty
              canvas.
            </p>
            <Link
              href="https://github.com/macg4dave/Roller_hoops/blob/main/docs/network_map/network_map_ideas.md"
              className="mapInspectorLink"
              target="_blank"
              rel="noreferrer"
            >
              Preview the mock contract
            </Link>
          </div>
        </section>
      </section>
    </div>
  );
}
