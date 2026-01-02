import Link from 'next/link';
import { redirect } from 'next/navigation';
import { randomUUID } from 'crypto';

import { Alert } from '@/app/_components/ui/Alert';
import { Button } from '@/app/_components/ui/Button';
import { EmptyState } from '@/app/_components/ui/EmptyState';
import { Field, Hint, Label } from '@/app/_components/ui/Field';
import { Input, Select } from '@/app/_components/ui/Inputs';
import type { components } from '@/lib/api-types';

import { MapCanvas } from './MapCanvas';
import { MapInspectorDetails } from './MapInspectorDetails';
import { MapPollingControls } from './MapPollingControls';
import { MapProjectionProvider } from './MapProjectionContext';
import { MapSelectionProvider } from './MapSelectionContext';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];
type MapProjection = components['schemas']['MapProjection'];

const LAYER_OPTIONS = [
  { id: 'physical', label: 'Physical', description: 'Cables, racks, and adjacency' },
  { id: 'l2', label: 'L2 (VLANs)', description: 'VLAN grouping (PVID only)' },
  { id: 'l3', label: 'L3 (Subnets)', description: 'Subnets and device membership' },
  { id: 'services', label: 'Services', description: 'Discovered ports and protocol services' },
  { id: 'security', label: 'Security', description: 'Zones, policies, and focus-driven flows' }
] as const satisfies ReadonlyArray<{ id: MapLayer; label: string; description: string }>;

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
    canvasHint: 'Physical adjacency projection',
    emptyTitle: 'Pick a focus to render physical adjacency'
  },
  l2: {
    canvasHint: 'L2 VLAN projection (PVID only)',
    emptyTitle: 'Pick a focus to render VLAN membership'
  },
  l3: {
    canvasHint: 'L3 subnet projection',
    emptyTitle: 'Pick a focus to render subnet membership'
  },
  services: {
    canvasHint: 'Services projection',
    emptyTitle: 'Pick a focus to render discovered services'
  },
  security: {
    canvasHint: 'Security projection',
    emptyTitle: 'Pick a focus to render security zones'
  }
};

const LAYER_FOCUS_SUPPORT = {
  physical: ['device'],
  l2: ['device', 'vlan'],
  l3: ['device', 'subnet'],
  services: ['device', 'service'],
  security: ['device', 'zone']
} as const satisfies Record<LayerId, readonly FocusType[]>;

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

  let projection: MapProjection | undefined;
  let projectionError: string | undefined;

  if (activeLayerId && focus && !focusWarning && !unknownLayer) {
    const base = process.env.CORE_GO_BASE_URL ?? 'http://localhost:8081';
    const params = new URLSearchParams();
    params.set('focusType', focus.type);
    params.set('focusId', focus.id);

    try {
      const res = await fetch(`${base}/api/v1/map/${activeLayerId}?${params.toString()}`, {
        cache: 'no-store',
        headers: {
          Accept: 'application/json',
          'X-Request-ID': randomUUID()
        }
      });

      if (res.ok) {
        projection = (await res.json()) as MapProjection;
      } else {
        const payload = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
        projectionError = payload?.error?.message ? payload.error.message : `Failed to load projection (${res.status}).`;
      }
    } catch (err) {
      projectionError = err instanceof Error ? err.message : 'Failed to load projection.';
    }
  }

  const clearFocusParams = new URLSearchParams(currentParams);
  clearFocusParams.delete('focusType');
  clearFocusParams.delete('focusId');

  const selectionScopeKey = activeLayerId
    ? `${activeLayerId}:${focus?.type ?? 'none'}:${focus?.id ?? 'none'}`
    : unknownLayer
      ? `unknown:${unknownLayer}`
      : 'no-layer';

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
              if (focus && !LAYER_FOCUS_SUPPORT[layer.id].includes(focus.type)) {
                nextParams.delete('focusType');
                nextParams.delete('focusId');
              }
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

        <MapSelectionProvider key={selectionScopeKey}>
          <MapProjectionProvider layer={activeLayerId} focus={focus} initialProjection={projection}>
            <section className="mapPanel mapCanvasPanel">
              <div className="mapCanvasHeader">
                <div className="mapCanvasHeaderText">
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
                {focus && projection && activeLayerId && !unknownLayer ? <MapPollingControls /> : null}
              </div>
              <div className="mapCanvasBody">
                {focus && projection && activeLayerId && !unknownLayer ? (
                  <MapCanvas projection={projection} activeLayerId={activeLayerId} currentParams={currentParams.toString()} />
                ) : (
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
                        {projectionError ? (
                          <p>Projection failed to load. The inspector shows the error response for this focus.</p>
                        ) : (
                          <p>Projection unavailable.</p>
                        )}
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
                )}
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
                {projectionError ? <Alert tone="warning">{projectionError}</Alert> : null}

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
                    <Hint>
                      Device focus ids are UUIDs. Subnet focus ids are CIDR strings (e.g. 10.0.1.0/24). VLAN focus ids are
                      integers (1-4094).
                    </Hint>
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

              <MapInspectorDetails
                focus={focus}
                focusTypeLabel={focusTypeLabel}
                projection={projection}
                activeLayerId={(activeLayerId ?? DEFAULT_LAYER) as MapLayer}
                currentParams={currentParams.toString()}
              />

              <div className="mapInspectorSection">
                <div className="mapInspectorHeading">Guidance</div>
                {projection?.guidance ? (
                  <p className="mapInspectorHint">{projection.guidance}</p>
                ) : (
                  <p className="mapInspectorHint">
                    URL-driven state: `/map?layer=l3&focusType=device&focusId=...`. No focus is valid and renders an empty
                    canvas.
                  </p>
                )}
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
          </MapProjectionProvider>
        </MapSelectionProvider>
      </section>
    </div>
  );
}
