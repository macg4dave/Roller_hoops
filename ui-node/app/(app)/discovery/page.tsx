import { DiscoveryPanel } from '../devices/DiscoveryPanel';
import type { DiscoveryStatus } from '../devices/types';
import { getSessionUser } from '../../../lib/auth/session';
import { fetchDiscoveryRuns, fetchDiscoveryStatus } from './api';
import { DiscoveryRunList } from './DiscoveryRunList';

export default async function DiscoveryPage() {
  const currentUser = await getSessionUser();
  const discoveryStatus = await fetchDiscoveryStatus();
  const readOnly = currentUser?.role === 'read-only';
  let discoveryRuns;
  let runsError: string | undefined;

  try {
    discoveryRuns = await fetchDiscoveryRuns(8);
  } catch (error) {
    runsError = error instanceof Error ? error.message : 'Unable to load discovery runs.';
    discoveryRuns = { runs: [], cursor: null };
  }

  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Discovery</h1>
        <p className="pageSubTitle">Run discovery, monitor progress, and debug failures.</p>
        <p className="hint">
          Discovery runs include the recent history of scoped sweeps and make it easy to inspect logs, errors, and completion status.
        </p>
      </header>

      <DiscoveryPanel status={discoveryStatus} readOnly={readOnly} />
      <DiscoveryRunList initialPage={discoveryRuns} errorMessage={runsError} />
    </section>
  );
}
