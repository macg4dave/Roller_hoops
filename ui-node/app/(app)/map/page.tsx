import { Card, CardBody } from '../../_components/ui/Card';
import { EmptyState } from '../../_components/ui/EmptyState';

export default async function MapPage() {
  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Map</h1>
        <p className="pageSubTitle">Layered explorer (Phase 13). Object-first, no spaghetti.</p>
      </header>

      <Card>
        <CardBody>
          <EmptyState title="Coming soon">
            The map shell (3-pane layout: layers / canvas / inspector) lands in Phase 13.
          </EmptyState>
        </CardBody>
      </Card>
    </section>
  );
}
