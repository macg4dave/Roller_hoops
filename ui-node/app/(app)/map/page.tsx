import Link from 'next/link';

import { Button } from '@/app/_components/ui/Button';
import { EmptyState } from '@/app/_components/ui/EmptyState';

const LAYER_OPTIONS = [
  { id: 'physical', label: 'Physical', description: 'Cables, racks, and adjacency' },
  { id: 'l2', label: 'L2 (VLANs)', description: 'VLAN grouping and tagged ports' },
  { id: 'l3', label: 'L3 (Subnets)', description: 'Subnets and device membership' },
  { id: 'services', label: 'Services', description: 'Discovered ports and protocol services' },
  { id: 'security', label: 'Security', description: 'Zones, policies, and focus-driven flows' }
] as const;

const RELATIONSHIP_ACTIONS = ['View in L3', 'View in Physical', 'View in Services'] as const;

const DEFAULT_LAYER = 'l3';

export default function MapPage() {
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
            {LAYER_OPTIONS.map((layer) => {
              const active = layer.id === DEFAULT_LAYER;
              return (
                <button
                  key={layer.id}
                  type="button"
                  className={`mapLayerItem${active ? ' mapLayerItemActive' : ''}`}
                  aria-current={active ? 'true' : undefined}
                >
                  <span className="mapLayerItemLabel">{layer.label}</span>
                  <span className="mapLayerItemMeta">{layer.description}</span>
                </button>
              );
            })}
          </div>
        </aside>

        <section className="mapPanel mapCanvasPanel">
          <div className="mapCanvasHeader">
            <p className="mapCanvasIntro">Canvas</p>
            <p className="mapCanvasHint">No focus selected yet.</p>
          </div>
          <div className="mapCanvasBody">
            <EmptyState title="Select a layer and focus to get started">
              <p>
                The canvas stays empty until you pick something to focus on. This keeps the view intentional and
                avoids the spaghetti effect from the mocks.
              </p>
              <p>
                Use the inspector on the right to jump between layers and follow relationships without losing context.
              </p>
            </EmptyState>
          </div>
        </section>

        <section className="mapPanel mapInspectorPanel">
          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Identity</div>
            <p className="mapInspectorValue">No object selected</p>
            <p className="mapInspectorHint">When focused, we will show identifiers, ownership, and metadata.</p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Status</div>
            <p className="mapInspectorValue">Awaiting focus</p>
            <p className="mapInspectorHint">Active focus will show health, last discovery time, and notes.</p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Relationships</div>
            <div className="mapInspectorActions">
              {RELATIONSHIP_ACTIONS.map((label) => (
                <Button key={label} variant="default" disabled>
                  {label}
                </Button>
              ))}
            </div>
            <p className="mapInspectorHint">
              Relationship actions will switch layers while keeping the focused object in view.
            </p>
          </div>

          <div className="mapInspectorSection">
            <div className="mapInspectorHeading">Guidance</div>
            <p className="mapInspectorHint">
              Deep-linking will ship in Phase 13.2. For now, explore each layer manually and follow the persona
              outlined in the docs.
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
